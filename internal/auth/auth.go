package auth

import (
	"database/sql"
	"log"
	"strings"

	"github.com/BigBr41n/GFTP-server/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type SQLiteAuth struct {
	db *sql.DB
}

type User struct {
	ID       int
	Username string
	FTPRoot  string
}

type Authenticator interface {
	Authenticate(username, password string) (*User, bool)
	Close() error
}

func NewAuthenticator(cfg config.Config) (Authenticator, error) {
	db, err := sql.Open("sqlite3", cfg.DBpath)
	if err != nil {
		return nil, err
	}

	// Verify the connection works
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &SQLiteAuth{db: db}, nil
}

// Close method for SQLiteAuth to close the database connection
func (sq *SQLiteAuth) Close() error {
	return sq.db.Close()
}

// Authenticate checks if the username and password match the stored credentials
func (sq *SQLiteAuth) Authenticate(username, password string) (*User, bool) {
	var user User
	var hashedPassword string

	// Trim whitespace from the username and password to prevent errors
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	// Fetch the hashed password from the database
	err := sq.db.QueryRow("SELECT id, username, password, ftpRoot FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &hashedPassword, &user.FTPRoot)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("User does not exist.")
			return nil, false
		}
		log.Printf("Database error: %v", err)
		return nil, false
	}

	// Compare the provided password with the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, false
	}

	// Authentication successful
	return &user, true
}
