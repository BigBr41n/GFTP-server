package ftp

import (
	"log"
	"net"

	"github.com/bigbr41n/GFTP-server/internal/auth"
	"github.com/bigbr41n/GFTP-server/internal/config"
)

type Server struct {
	config *config.Config
	auth   auth.Authenticator // Use pointer to interface
}

func NewServer(cfg *config.Config) *Server {
	// Create a new server with the provided configuration
	auth, err := auth.NewAuthenticator(*cfg)
	if err != nil {
		log.Fatalf("Error creating authenticator: %v", err)
	}

	return &Server{
		config: cfg,
		auth:   auth,
	}
}

func (s *Server) ListenAndServe() error {
	// Create a new listener on the specified port that came with the config
	ln, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close() 

	// Start accepting connections from clients
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// Function to handle connections
func (s *Server) handleConnection(conn net.Conn) {
	session := NewSession(conn, s.config, s.auth)
	session.Server()
}
