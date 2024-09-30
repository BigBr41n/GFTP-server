package ftp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type FTPuser struct {
	conn      net.Conn
	Username  string
	FTPRoot   string
	currentDir string
}

type FTPhandler interface {
	HandleCommands()
}

func NewCommandsHandler(FTPRoot, username string, conn net.Conn) FTPhandler {
	absRoot, err := filepath.Abs(FTPRoot)
	if err != nil {
		absRoot = FTPRoot 
	}
	return &FTPuser{
		conn:     conn,
		Username: username,
		FTPRoot:  absRoot,
		currentDir: absRoot,
	}
}

func (ftu *FTPuser) HandleCommands() {
	//change the dir to user dir
	err := os.Chdir(ftu.FTPRoot)
    if err!= nil {
        ftu.writeResponse(fmt.Sprintf("Error changing directory: %v\r\n", err))
        return
    }
	for {
		// Read the command from the client
		command, err := ftu.readInput()
		if err != nil {
			ftu.writeResponse("500 Internal server error.\r\n")
			return
		}

		// Process the command
		switch {
		case strings.HasPrefix(command, "LS"):
			ftu.handleLS()
		case strings.HasPrefix(command, "CD"):
			ftu.handleCD(strings.TrimSpace(command[2:]))
		case strings.HasPrefix(command, "RM"):
			ftu.handleRM(strings.TrimSpace(command[2:]))
		case strings.HasPrefix(command, "PUT"):
			ftu.handlePUT(strings.TrimSpace(command[3:]))
		case strings.HasPrefix(command, "GET"):
			ftu.handleGET(strings.TrimSpace(command[3:]))
		case strings.HasPrefix(command, "PWD"):
			ftu.handlePWD()
		case strings.HasPrefix(command, "QUIT"):
			ftu.writeResponse("221 Goodbye.\r\n")
			ftu.conn.Close()
            return
		default:
			ftu.writeResponse("500 Unknown command.\r\n")
		}
	}
}

// handleLS handles the LS (list directory) command
func (ftu *FTPuser) handleLS() {
	ftu.writeResponse("Listing files...\r\n")
	files, err := os.ReadDir(ftu.currentDir)
	if err != nil {
		ftu.writeResponse(fmt.Sprintf("Error reading directory: %v\r\n", err))
		return
	}
	//check if no files exist
	if len(files) == 0 {
        ftu.writeResponse("No files found in this directory.\r\n")
        return
    }
	for _, file := range files {
		ftu.writeResponse(file.Name() + "\r\n")
	}
}



// handleCD handles the CD (change directory) command
func (ftu *FTPuser) handleCD(path string) {
    // Resolve the new directory path
    newPath := filepath.Join(ftu.currentDir, path)
    absPath, err := filepath.Abs(newPath)
    if err != nil {
        ftu.writeResponse("500 Invalid directory.\r\n")
        return
    }

    // Ensure the new path is within the user's FTP root directory
    if !strings.HasPrefix(absPath, ftu.FTPRoot) {
        ftu.writeResponse("550 Access denied.\r\n")
        return
    }

    // Check if the new directory exists and is indeed a directory
    stat, err := os.Stat(absPath)
    if os.IsNotExist(err) || !stat.IsDir() {
        ftu.writeResponse("550 Directory does not exist or is not a directory.\r\n")
        return
    }

	if err := os.Chdir(absPath); err != nil {
		ftu.writeResponse("500 Failed to change directory.\r\n")
		return
	}

    // Change directory
    ftu.currentDir = absPath
    relativePath, err := filepath.Rel(ftu.FTPRoot, absPath)
    if err != nil {
        ftu.writeResponse("500 Failed to determine relative path.\r\n")
        return
    }

    // Ensure the relative path always starts with a slash
    if !strings.HasPrefix(relativePath, "/") {
        relativePath = "/" + relativePath
    }

    // Respond with the new relative path
    ftu.writeResponse(fmt.Sprintf("250 Directory changed to %s\r\n", relativePath))
}




// handlePWD handles the PWD (Print Working Directory) command
func (ftu *FTPuser) handlePWD() {
    // Use ftu.currentDir instead of os.Getwd()
    relativePath, err := filepath.Rel(ftu.FTPRoot, ftu.currentDir)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error getting relative path: %v\r\n", err))
        return
    }

    // Ensure the path always starts with a slash
    if !strings.HasPrefix(relativePath, "/") {
        relativePath = "/" + relativePath
    }

    // If the relative path is empty, it means we're at the root
    if relativePath == "/" {
        relativePath = "/" + ftu.Username
    }

    ftu.writeResponse(fmt.Sprintf("257 \"%s\" is the current directory\r\n", relativePath))
}




// handleRM handles the RM (remove file) command
func (ftu *FTPuser) handleRM(path string) {
	fullPath := filepath.Join(ftu.currentDir, path)
	err := os.Remove(fullPath)
	if err != nil {
		ftu.writeResponse(fmt.Sprintf("Error removing file: %v\r\n", err))
		return
	}
	ftu.writeResponse(fmt.Sprintf("File %s removed.\r\n", path))
}

// handlePUT handles the PUT (upload file) command
func (ftu *FTPuser) handlePUT(filename string) {
	ftu.writeResponse(fmt.Sprintf("Ready to receive file %s...\r\n", filename))

	file, err := os.Create(filepath.Join(ftu.currentDir, filename))
	if err != nil {
		ftu.writeResponse(fmt.Sprintf("Error creating file: %v\r\n", err))
		return
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := ftu.conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			ftu.writeResponse(fmt.Sprintf("Error receiving file: %v\r\n", err))
			return
		}
		if _, err := file.Write(buffer[:n]); err != nil {
			ftu.writeResponse(fmt.Sprintf("Error writing file: %v\r\n", err))
			return
		}
	}
	ftu.writeResponse("File upload complete.\r\n")
}



// handleGET handles the GET (download file) command
func (ftu *FTPuser) handleGET(filename string) {
    fullPath := filepath.Join(ftu.currentDir, filename)


	//the var of the file size 
	var size int64
    
    // Check if file exists and get its size
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            ftu.writeResponse("550 File not found.\r\n")
        } else {
            ftu.writeResponse(fmt.Sprintf("550 Error accessing file: %v\r\n", err))
        }
        return
    }


	//get the size of the file 
	size = fileInfo.Size()


	//send the size to the client 
	binary.Write(ftu.conn, binary.LittleEndian, size);

    // Open the file
    file, err := os.Open(fullPath)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error opening file: %v\r\n", err))
        return
    }
    defer file.Close()

    // Inform the client that the file transfer is about to begin
    ftu.writeResponse(fmt.Sprintf("150 Opening BINARY mode data connection for %s (%d bytes).\r\n", filename, fileInfo.Size()))

    // Send the file content
    _, err = io.CopyN(ftu.conn, file, size) // copy the exact size from the file (the file size itself)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error sending file: %v\r\n", err))
        return
    }

    ftu.writeResponse("\n 226 Transfer complete.\r\n")
}

func (ftu *FTPuser) readInput() (string, error) {
	buffer := make([]byte, 1024)
	n, err := ftu.conn.Read(buffer)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buffer[:n])), nil
}

func (ftu *FTPuser) writeResponse(res string) error {
	_, err := ftu.conn.Write([]byte(res))
	return err
}
