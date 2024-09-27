package ftp

import (
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
}

type FTPhandler interface {
	HandleCommands()
}

func NewCommandsHandler(FTPRoot, username string, conn net.Conn) FTPhandler {
	absRoot, err := filepath.Abs(FTPRoot)
	if err != nil {
		absRoot = FTPRoot // fallback to the given path
	}
	return &FTPuser{
		conn:     conn,
		Username: username,
		FTPRoot:  absRoot,
	}
}

func (ftu *FTPuser) HandleCommands() {
	for {
		// Read the command from the client
		command, err := ftu.readInput("ftp> ")
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
		default:
			ftu.writeResponse("500 Unknown command.\r\n")
		}
	}
}

// handleLS handles the LS (list directory) command
func (ftu *FTPuser) handleLS() {
	ftu.writeResponse("Listing files...\r\n")
	files, err := os.ReadDir(ftu.FTPRoot)
	if err != nil {
		ftu.writeResponse(fmt.Sprintf("Error reading directory: %v\r\n", err))
		return
	}
	for _, file := range files {
		ftu.writeResponse(file.Name() + "\r\n")
	}
}

// handleCD handles the CD (change directory) command
func (ftu *FTPuser) handleCD(path string) {
	// Resolve the new directory path
	newPath := filepath.Join(ftu.FTPRoot, path)
	absPath, err := filepath.Abs(newPath)
	if err != nil || !strings.HasPrefix(absPath, ftu.FTPRoot) {
		ftu.writeResponse("500 Invalid directory.\r\n")
		return
	}

	// Change directory
	ftu.FTPRoot = absPath
	ftu.writeResponse(fmt.Sprintf("Directory changed to %s\r\n", absPath))
}

// handleRM handles the RM (remove file) command
func (ftu *FTPuser) handleRM(path string) {
	fullPath := filepath.Join(ftu.FTPRoot, path)
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

	file, err := os.Create(filepath.Join(ftu.FTPRoot, filename))
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
	fullPath := filepath.Join(ftu.FTPRoot, filename)
	file, err := os.Open(fullPath)
	if err != nil {
		ftu.writeResponse(fmt.Sprintf("Error opening file: %v\r\n", err))
		return
	}
	defer file.Close()

	ftu.writeResponse(fmt.Sprintf("Starting file download: %s\r\n", filename))

	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			ftu.writeResponse(fmt.Sprintf("Error reading file: %v\r\n", err))
			return
		}
		if _, err := ftu.conn.Write(buffer[:n]); err != nil {
			ftu.writeResponse(fmt.Sprintf("Error sending file: %v\r\n", err))
			return
		}
	}
	ftu.writeResponse("File download complete.\r\n")
}

func (ftu *FTPuser) readInput(prompt string) (string, error) {
	if err := ftu.writeResponse(prompt); err != nil {
		return "", err
	}

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
