package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// RegisterHandler handles user registration requests.
// POST /api/v1/auth/register
func RegisterHandler(store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req model.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		cleanEmail := strings.TrimSpace(req.Email)
		cleanPassword := req.Password
		cleanName := strings.TrimSpace(req.Name)

		if !auth.ValidateEmail(cleanEmail) {
			writeJSONError(w, http.StatusBadRequest, "Invalid email address format")
			return
		}

		if len(cleanPassword) < auth.MinPasswordLength {
			writeJSONError(w, http.StatusBadRequest, "Password must be at least 8 characters long")
			return
		}

		if cleanName == "" {
			parts := strings.Split(cleanEmail, "@")
			cleanName = parts[0]
		}

		hashedPassword, err := auth.HashPassword(cleanPassword)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		user := &model.User{
			Email:        cleanEmail,
			PasswordHash: hashedPassword,
			Name:         cleanName,
		}

		createdUser, err := store.CreateUser(r.Context(), user)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicateEmail) {
				writeJSONError(w, http.StatusConflict, "Email address already registered")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to create user account")
			return
		}

		// Create session upon registration
		token, err := auth.GenerateSecureToken(auth.DefaultTokenBytes)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate session token")
			return
		}

		expiresAt := time.Now().UTC().Add(auth.SessionDuration)
		session := &model.Session{
			Token:     token,
			UserID:    createdUser.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
		}

		if err := store.CreateSession(r.Context(), session); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to persist user session")
			return
		}

		secure := isSecureRequest(r)
		http.SetCookie(w, auth.NewSessionCookie(token, expiresAt, secure))

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "User registered successfully",
			"user":    createdUser.ToProfile(),
		})
	}
}

// LoginHandler handles user authentication requests.
// POST /api/v1/auth/login
func LoginHandler(store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req model.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		cleanEmail := strings.TrimSpace(req.Email)
		if cleanEmail == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		user, err := store.GetUserByEmail(r.Context(), cleanEmail)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				writeJSONError(w, http.StatusUnauthorized, "Invalid email or password")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to query user")
			return
		}

		if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		token, err := auth.GenerateSecureToken(auth.DefaultTokenBytes)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate session token")
			return
		}

		expiresAt := time.Now().UTC().Add(auth.SessionDuration)
		session := &model.Session{
			Token:     token,
			UserID:    user.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
		}

		if err := store.CreateSession(r.Context(), session); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to persist user session")
			return
		}

		secure := isSecureRequest(r)
		http.SetCookie(w, auth.NewSessionCookie(token, expiresAt, secure))

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Login successful",
			"user":    user.ToProfile(),
		})
	}
}

// LogoutHandler handles session termination requests.
// POST /api/v1/auth/logout
func LogoutHandler(store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		cookie, err := r.Cookie(auth.SessionCookieName)
		if err == nil && cookie.Value != "" {
			_ = store.DeleteSession(r.Context(), cookie.Value)
		}

		http.SetCookie(w, auth.ClearSessionCookie())

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Logged out successfully",
		})
	}
}

// MeHandler returns the sanitized profile of the currently authenticated user.
// GET /api/v1/auth/me
func MeHandler(store *storage.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		user, ok := UserFromContext(r.Context())
		if !ok {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			var err error
			user, err = store.GetUserByID(r.Context(), userID)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "User account not found")
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"user": user.ToProfile(),
		})
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
