package ftp

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type FTPuser struct {
	conn      		net.Conn
	Username  		string
	userFTPRoot  	string
	serverFTPRoot  	string
	currentDir 		string
}

type FTPhandler interface {
	HandleCommands()
}

func NewCommandsHandler(userFTPRoot,serverFtpRoot,username string, conn net.Conn) FTPhandler {
	return &FTPuser{
		conn:     conn,
		Username: username,
		userFTPRoot:  userFTPRoot,
		serverFTPRoot: serverFtpRoot,
		currentDir: userFTPRoot,
	}
}

func (ftu *FTPuser) HandleCommands() {
	defer ftu.conn.Close()

	//clean up after disconnect
	defer func(){
		err := os.Chdir(ftu.serverFTPRoot)
        if err!= nil {
            fmt.Printf("Error: %v\n", err)
        }
	}()

	//change the dir to the user dir 
	err := os.Chdir(ftu.userFTPRoot)
	if err != nil {
		ftu.writeResponse("Error changing directory")
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
        case strings.HasPrefix(command, "MKDIR"):
            ftu.handleMkdir(strings.TrimSpace(command[5:]))
        case strings.HasPrefix(command, "DRM"):
            ftu.handleDRM(strings.TrimSpace(command[3:]))
		case strings.HasPrefix(command, "QUIT"):
			ftu.writeResponse("221 Goodbye.\r\n")
			ftu.conn.Close()
            return
		default:
			ftu.writeResponse("500 Unknown command.\r\n")
		}
	}
}



// handleLS handles the LIST command (commonly used for directory listing)
func (ftu *FTPuser) handleLS() {
    // Read directory contents
    files, err := os.ReadDir(ftu.currentDir)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error reading directory: %v\r\n", err))
        return
    }

    // Prepare the listing
    var listing strings.Builder
    for _, file := range files {
        info, err := file.Info()
        if err != nil {
            log.Printf("Error getting file info for %s: %v", file.Name(), err)
            continue
        }
        listing.WriteString(formatFileInfo(info) + "\r\n")
    }

    // Send the listing
    if _, err := ftu.conn.Write([]byte(listing.String())); err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error sending directory listing: %v\r\n", err))
        return
    }
}

func formatFileInfo(info os.FileInfo) string {
    // Format: <file mode> <number of links> <owner name> <group name> <file size> <time of last modification> <file/dir name>
    return fmt.Sprintf("%s %8d %s %s",
        info.Mode().String(),
        info.Size(),
        info.ModTime().Format("Jan _2 15:04"),
        info.Name())
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
    if !strings.HasPrefix(absPath, ftu.userFTPRoot) {
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
    relativePath, err := filepath.Rel(ftu.userFTPRoot, absPath)
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
    relativePath, err := filepath.Rel(ftu.userFTPRoot, ftu.currentDir)
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
    // Create and open the file for writing
    file, err := os.Create(filename)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("Error creating file: %v\r\n", err))
        fmt.Println("the error  \n:", err);
        return
    }
    defer file.Close()
    ftu.writeResponse("File Created.\r\n")

    //// Read the file size from the client
    var size int64 
    err = binary.Read(ftu.conn, binary.LittleEndian, &size)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("Error reading file size: %v\r\n", err))
        return
    }
    ftu.writeResponse("Send file\r\n")
    

    // Read the exact size of the file from the client
    _, err = io.CopyN(file, ftu.conn, size)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("Error receiving file: %v\r\n", err))
        return
    }

    ftu.writeResponse(fmt.Sprintf("File %s uploaded successfully.\r\n", filename))
    fmt.Printf("File uploaded to: %s/%s\n", ftu.currentDir,filename)
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


    // Send the file content
    _, err = io.CopyN(ftu.conn, file, size) // copy the exact size from the file (the file size itself)
    if err != nil {
        ftu.writeResponse(fmt.Sprintf("550 Error sending file: %v\r\n", err))
        return
    }
}


// handle mkdir command to create a new folder (sub directory) in the current directory
func (ftu * FTPuser) handleMkdir(folderName string) {
    // Get all the files in the current directory 
    files , err := os.ReadDir(ftu.currentDir)
    if err != nil {
        ftu.writeResponse("550 Error while creating a new directory ")
        return 
    }

    // Check if there is a directory with the same name already
    for _, file := range files {
        if file.Name() == folderName && file.IsDir() {
            ftu.writeResponse(fmt.Sprintf("Directory %s already exists", folderName))
            return 
        }
    }

    // If no error create a new folder in the dir 
    err = os.Mkdir(filepath.Join(ftu.currentDir, folderName), 0755)
    if err!= nil {
        ftu.writeResponse("550 Error while creating a new directory ")
        return 
    }

    // Return a feedback to the user
    ftu.writeResponse(fmt.Sprintf("Directory %s created successfully", folderName))
}



//handle DRM to delete a directory and its content
func (ftu *FTPuser) handleDRM(folderName string) {
    // Get all the files and directories in the current directory 
    files , err := os.ReadDir(ftu.currentDir)
    if err!= nil {
        ftu.writeResponse("550 Error while deleting a directory ")
        return 
    }

    // Check if there is a directory with the same name already
    for _, file := range files {
        if file.Name() == folderName && file.IsDir() {
            // Delete the directory and its content recursively
            err = os.RemoveAll(filepath.Join(ftu.currentDir, folderName))
            if err!= nil {
                ftu.writeResponse("550 Error while deleting a directory ")
                return 
            }

            // Return a feedback to the user
            ftu.writeResponse(fmt.Sprintf("Directory %s and its content deleted successfully", folderName))
            return 
        }
    }
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
