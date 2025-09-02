package webui

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	ID        string
	CreatedAt time.Time
	LastSeen  time.Time
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
	ttl      time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(ttl time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go sm.cleanup()

	return sm
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession() string {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate random session ID
	bytes := make([]byte, 16)
	rand.Read(bytes)
	sessionID := hex.EncodeToString(bytes)

	// Create session
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	sm.sessions[sessionID] = session
	return sessionID
}

// ValidateSession validates a session and updates last seen time
func (sm *SessionManager) ValidateSession(sessionID string) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false
	}

	// Check if session has expired
	if time.Since(session.LastSeen) > sm.ttl {
		delete(sm.sessions, sessionID)
		return false
	}

	// Update last seen time
	session.LastSeen = time.Now()
	return true
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.sessions, sessionID)
}

// cleanup removes expired sessions
func (sm *SessionManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.Sub(session.LastSeen) > sm.ttl {
				delete(sm.sessions, id)
			}
		}
		sm.mutex.Unlock()
	}
}

// AuthMiddleware provides authentication for WebUI
type AuthMiddleware struct {
	password       string
	sessionManager *SessionManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(password string) *AuthMiddleware {
	return &AuthMiddleware{
		password:       password,
		sessionManager: NewSessionManager(24 * time.Hour), // 24 hour session
	}
}

// UpdateConfig updates the auth middleware configuration
func (am *AuthMiddleware) UpdateConfig(password string) {
	am.password = password
}

// RequireAuth checks if authentication is required and validates session
func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no password is set, no authentication required
		if am.password == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for session cookie
		cookie, err := r.Cookie("webui_session")
		if err != nil || !am.sessionManager.ValidateSession(cookie.Value) {
			// Redirect to login page
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// HandleLogin handles login requests
func (am *AuthMiddleware) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if am.password == "" {
		// No authentication required, redirect to main page
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Show login page
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(loginHTML))
		return
	}

	if r.Method == "POST" {
		// Process login
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")
		if password != am.password {
			// Show login page with error
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(loginHTMLWithError))
			return
		}

		// Create session
		sessionID := am.sessionManager.CreateSession()

		// Set session cookie
		cookie := &http.Cookie{
			Name:     "webui_session",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true if using HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 60 * 60, // 24 hours
		}
		http.SetCookie(w, cookie)

		// Redirect to main page
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleLogout handles logout requests
func (am *AuthMiddleware) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("webui_session")
	if err == nil {
		// Delete session
		am.sessionManager.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	clearCookie := &http.Cookie{
		Name:     "webui_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	}
	http.SetCookie(w, clearCookie)

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusFound)
}
