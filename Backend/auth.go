package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SessionManager manages user sessions
type SessionManager struct {
	db *sql.DB
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{
		db: db,
	}
}

// Create a session for the user
func (sm *SessionManager) CreateSession(w http.ResponseWriter, userID int) (string, error) {
	// Generate session ID
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	sessionID := base64.URLEncoding.EncodeToString(b)

	// Set expiration time (24 hours)
	expiry := time.Now().Add(24 * time.Hour)

	// Save session to database
	err = saveSession(sm.db, sessionID, userID, expiry)
	if err != nil {
		return "", err
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		MaxAge:   int(24 * time.Hour.Seconds()),
	})

	return sessionID, nil
}

// Get the current logged-in user
func (sm *SessionManager) GetLoggedInUser(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, nil // No cookie, not logged in
	}

	userID, err := getUserIDFromSession(sm.db, cookie.Value)
	if err != nil {
		return nil, err
	}

	if userID == 0 {
		return nil, nil // Session not found or expired
	}

	return getUserByID(sm.db, userID)
}

// Check if a user is logged in
func (sm *SessionManager) IsLoggedIn(r *http.Request) bool {
	user, err := sm.GetLoggedInUser(r)
	if err != nil {
		log.Printf("Error checking if user is logged in: %v", err)
		return false
	}
	return user != nil
}

// Get API key from session
func (sm *SessionManager) ApiKeyFromRequest(r *http.Request) string {
	user, err := sm.GetLoggedInUser(r)
	if err != nil || user == nil {
		return ""
	}
	return user.APIKey
}

// Login handler handles user login
func (sm *SessionManager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Only handle POST requests for actual login
	if r.Method == "POST" {
		// Parse form data
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		apiKey := r.FormValue("api_key")

		if username == "" || apiKey == "" {
			http.Error(w, "Username and API key are required", http.StatusBadRequest)
			return
		}

		// Create or update user
		user, err := createOrUpdateUser(sm.db, username, apiKey)
		if err != nil {
			log.Printf("Error creating/updating user: %v", err)
			http.Error(w, "Failed to process user data", http.StatusInternalServerError)
			return
		}

		// Create a session
		_, err = sm.CreateSession(w, user.ID)
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Redirect to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// For GET requests, serve the login page
	http.ServeFile(w, r, "../static/login.html")
}

// Logout handler terminates the user session
func (sm *SessionManager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete session from database
		deleteSession(sm.db, cookie.Value)
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Middleware to check if user is authenticated
func (sm *SessionManager) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sm.IsLoggedIn(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// Update user's money
func (sm *SessionManager) UpdateUserMoney(userID int, amount int) error {
	// Get current user
	user, err := getUserByID(sm.db, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Update money
	newMoney := user.Money + amount
	return updateUserMoney(sm.db, userID, newMoney)
}

// Get user's current money
func (sm *SessionManager) GetUserMoney(userID int) (int, error) {
	user, err := getUserByID(sm.db, userID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, fmt.Errorf("user not found")
	}

	return user.Money, nil
}
