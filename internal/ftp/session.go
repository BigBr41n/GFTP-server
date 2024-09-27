package ftp

import (
	"net"

	"github.com/bigbr41n/GFTP-server/internal/auth"
	"github.com/bigbr41n/GFTP-server/internal/config"
)

type Session struct {
	conn   net.Conn
	config *config.Config
	auth   auth.Authenticator
	user   *auth.User
}

func NewSession(con net.Conn, cfg *config.Config, auth auth.Authenticator) *Session {
	return &Session{
		conn:   con,
		config: cfg,
		auth:   auth,
	}
}

func (s *Session) Server() {
	defer s.conn.Close()

	// Read the username and password from the client
	username, err := s.readInput("ENTER USERNAME: \n")
	if err != nil {
		s.writeResponse("INTERNAL SERVER ERROR")
		return // Ensure we exit if there's an error
	}

	password, err := s.readInput("ENTER PASSWORD: \n")
	if err != nil {
		s.writeResponse("INTERNAL SERVER ERROR")
		return // Ensure we exit if there's an error
	}

	// Authenticate the user
	user, authenticated := s.auth.Authenticate(username, password)
	if authenticated {
		s.user = user
		s.conn.Write([]byte("230 User logged in, proceed.\r\n"))

		handler := NewCommandsHandler(s.user.FTPRoot , s.user.Username, s.conn)
		//handel all the commands received 
		handler.HandleCommands()
	} else {
		s.writeResponse("530 Login incorrect.")
	}
}

func (s *Session) readInput(prompt string) (string, error) {
	// Write the prompt to the client
	if err := s.writeResponse(prompt); err != nil {
		return "", err
	}

	// Create a buffer to store incoming data
	buffer := make([]byte, 1024) // Adjust size as needed

	// Read from the connection
	n, err := s.conn.Read(buffer)
	if err != nil {
		return "", err
	}

	// Return the received input, trimmed of whitespace
	return string(buffer[:n]), nil
}

func (s *Session) writeResponse(res string) error {
	_, err := s.conn.Write([]byte(res))
	return err
}
