package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// UserSession represents a user's authentication session
type UserSession struct {
	Username  string    `json:"username"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// SessionManager handles user sessions
type SessionManager struct {
	sessions      map[string]*UserSession
	mutex         sync.RWMutex
	sessionExpiry time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:      make(map[string]*UserSession),
		sessionExpiry: 24 * time.Hour, // Sessions expire after 24 hours
	}
}

// GenerateSessionID creates a secure random session ID
func (sm *SessionManager) GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSession creates a new user session
func (sm *SessionManager) CreateSession(username, apiKey string) (string, error) {
	sessionID, err := sm.GenerateSessionID()
	if err != nil {
		return "", err
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.sessions[sessionID] = &UserSession{
		Username:  username,
		APIKey:    apiKey,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	return sessionID, nil
}

// GetSession retrieves a user session by session ID
func (sm *SessionManager) GetSession(sessionID string) *UserSession {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil
	}

	// Check if session has expired
	if time.Since(session.LastUsed) > sm.sessionExpiry {
		sm.mutex.RUnlock()
		sm.DeleteSession(sessionID)
		sm.mutex.RLock()
		return nil
	}

	// Update last used time
	session.LastUsed = time.Now()
	return session
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.sessions, sessionID)
}

// Middleware to check if user is authenticated
func (sm *SessionManager) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check if session exists
		session := sm.GetSession(cookie.Value)
		if session == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Store session in request context
		ctx := context.WithValue(r.Context(), sessionKey{}, session)
		r = r.WithContext(ctx)

		// Call the next handler
		next(w, r)
	}
}

// LoginHandler handles user login
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

		// Create a new session
		sessionID, err := sm.CreateSession(username, apiKey)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil, // Set Secure flag if using HTTPS
			MaxAge:   int(sm.sessionExpiry.Seconds()),
		})

		// Redirect to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// For GET requests, serve the login page from the static directory
	http.ServeFile(w, r, "../static/login.html")
}

// LogoutHandler handles user logout
func (sm *SessionManager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete the session
		sm.DeleteSession(cookie.Value)
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   -1, // Delete the cookie
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ApiKeyFromRequest extracts the API key from the current session
func (sm *SessionManager) ApiKeyFromRequest(r *http.Request) string {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}

	// Get session
	session := sm.GetSession(cookie.Value)
	if session == nil {
		return ""
	}

	return session.APIKey
}

// JSON response for login API
func (sm *SessionManager) LoginAPIHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// Parse JSON body
	var credentials struct {
		Username string `json:"username"`
		APIKey   string `json:"api_key"`
	}

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if credentials.Username == "" || credentials.APIKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Username and API key are required"})
		return
	}

	// Create a new session
	sessionID, err := sm.CreateSession(credentials.Username, credentials.APIKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		MaxAge:   int(sm.sessionExpiry.Seconds()),
	})

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Login successful"})
}

// Helper type for request context
type sessionKey struct{}

// SessionFromContext gets the user session from a context
func SessionFromContext(ctx context.Context) (*UserSession, bool) {
	session, ok := ctx.Value(sessionKey{}).(*UserSession)
	return session, ok
}
