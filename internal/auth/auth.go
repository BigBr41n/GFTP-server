package auth

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/bigbr41n/GFTP-server/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type SQLiteAuth struct {
	db *sql.DB
}




type Authenticator interface {
	Authenticate(username, password string) bool
	CreateUser(username, password string) error
	DeleteUser(username string) error
	CheckUserExists(username string) (bool, error)
	Close() error
}



func NewAuthenticator(cfg config.Config) (*SQLiteAuth, error) {
	db, err := sql.Open("sqlite3", cfg.DBpath)
	if err != nil {
		return nil, err
	}

	// Verify the connection works
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &SQLiteAuth{db: db}, nil
}

// Close method for SQLiteAuth to close the database connection
func (sq *SQLiteAuth) Close() error {
	return sq.db.Close()
}




// Authenticate checks if the username and password match the stored credentials
func (sq *SQLiteAuth) Authenticate(username, password string) bool {
	var hashedPassword string

	// Fetch the hashed password from the database
	err := sq.db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = bcrypt.CompareHashAndPassword([]byte(""), []byte(password)) 
			log.Println("User does not exist.")
		} else {
			log.Printf("Database error: %v", err)
		}
		return false
	}

	// Compare the provided password with the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return false
	}

	// Authentication successful
	return true
}




// CreateUser adds a new user with a hashed password to the database
func (sq *SQLiteAuth) CreateUser(username, password string, cfg config.Config) error {
	exists, err := sq.CheckUserExists(username)
	if err != nil {
		return err
	}
	if exists {
		return sql.ErrNoRows // User already exists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Each user should have only one root and only accessible via the owner (created user) 
	ftpRoot, err := createUserDirectory(cfg.FTPRoot, username);
	if err != nil {
		return err;
	}

	_, err = sq.db.Exec("INSERT INTO users (username, password) VALUES (?, ?, ?)", username, hashedPassword, ftpRoot)
	return err
}


// DeleteUser removes a user from the database by username
func (sq *SQLiteAuth) DeleteUser(username string) error {
	_, err := sq.db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
	}
	return err
}



// CheckUserExists checks if a user exists in the database
func (sq *SQLiteAuth) CheckUserExists(username string) (bool, error) {
	var exists bool
	err := sq.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	return exists, err
}


// Function to create a user directory on the server
func createUserDirectory(FTPRoot ,username string) (string , error) {
	// Define the user directory path
	userDir := fmt.Sprintf("%s/%s",FTPRoot, username)

	// Create the directory for the user
	err := os.Mkdir(userDir, 0700) // Permissions set to 700
	if err != nil {
		return "", fmt.Errorf("failed to create directory for user %s: %w", username, err)
	}
	return userDir , nil
}

