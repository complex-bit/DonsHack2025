package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// User represents a user in the database
type User struct {
	ID        int
	Username  string
	APIKey    string
	Money     int
	CreatedAt time.Time
	LastLogin time.Time
}

// Initialize database connection and tables
func initDB() (*sql.DB, error) {
	// Create database if it doesn't exist
	dbPath := "./users.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Create users table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		api_key TEXT NOT NULL,
		money INTEGER DEFAULT 100,
		created_at DATETIME,
		last_login DATETIME
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER,
		created_at DATETIME,
		expires_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	return db, nil
}

// Create or update user in the database
func createOrUpdateUser(db *sql.DB, username, apiKey string) (*User, error) {
	// Check if user already exists
	var user User
	err := db.QueryRow("SELECT id, username, api_key, money, created_at, last_login FROM users WHERE username = ?", username).Scan(
		&user.ID, &user.Username, &user.APIKey, &user.Money, &user.CreatedAt, &user.LastLogin)

	if err == sql.ErrNoRows {
		// User doesn't exist, create new user
		now := time.Now()
		result, err := db.Exec("INSERT INTO users (username, api_key, money, created_at, last_login) VALUES (?, ?, ?, ?, ?)",
			username, apiKey, 100, now, now)
		if err != nil {
			return nil, fmt.Errorf("error creating user: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("error getting user ID: %v", err)
		}

		user = User{
			ID:        int(id),
			Username:  username,
			APIKey:    apiKey,
			Money:     100, // Default starting money
			CreatedAt: now,
			LastLogin: now,
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking for existing user: %v", err)
	} else {
		// User exists, update last login and API key
		now := time.Now()
		_, err := db.Exec("UPDATE users SET last_login = ?, api_key = ? WHERE id = ?",
			now, apiKey, user.ID)
		if err != nil {
			return nil, fmt.Errorf("error updating user: %v", err)
		}
		user.LastLogin = now
		user.APIKey = apiKey
	}

	return &user, nil
}

// Get user by ID
func getUserByID(db *sql.DB, id int) (*User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, api_key, money, created_at, last_login FROM users WHERE id = ?", id).Scan(
		&user.ID, &user.Username, &user.APIKey, &user.Money, &user.CreatedAt, &user.LastLogin)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	} else if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return &user, nil
}

// Get user by username
func getUserByUsername(db *sql.DB, username string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT id, username, api_key, money, created_at, last_login FROM users WHERE username = ?", username).Scan(
		&user.ID, &user.Username, &user.APIKey, &user.Money, &user.CreatedAt, &user.LastLogin)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	} else if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return &user, nil
}

// Update user's money
func updateUserMoney(db *sql.DB, userID int, newMoney int) error {
	_, err := db.Exec("UPDATE users SET money = ? WHERE id = ?", newMoney, userID)
	if err != nil {
		return fmt.Errorf("error updating user money: %v", err)
	}
	return nil
}

// Save session to database
func saveSession(db *sql.DB, sessionID string, userID int, expiry time.Time) error {
	_, err := db.Exec("INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		sessionID, userID, time.Now(), expiry)

	if err != nil {
		return fmt.Errorf("error saving session: %v", err)
	}

	return nil
}

// Get user ID from session
func getUserIDFromSession(db *sql.DB, sessionID string) (int, error) {
	var userID int
	var expiresAt time.Time

	err := db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE id = ?", sessionID).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return 0, nil // Session not found
	} else if err != nil {
		return 0, fmt.Errorf("error getting session: %v", err)
	}

	// Check if session has expired
	if time.Now().After(expiresAt) {
		// Delete expired session
		_, err = db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
		if err != nil {
			log.Printf("Error deleting expired session: %v", err)
		}
		return 0, nil
	}

	return userID, nil
}

// Delete session from database
func deleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("error deleting session: %v", err)
	}
	return nil
}
