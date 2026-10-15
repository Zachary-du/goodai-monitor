package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
	"github.com/Zachary-du/goodai-monitor/internal/config"
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

// ProvidersHandler returns enabled third-party identity providers.
// GET /api/v1/auth/providers
func ProvidersHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		isGoogleConfigured := cfg != nil && cfg.IsGoogleOAuthConfigured()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"google": isGoogleConfigured,
		})
	}
}

// GoogleLoginHandler initiates the Google OAuth 2.0 Single Sign-On handshake.
// GET /api/v1/auth/google/login
func GoogleLoginHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if cfg == nil || !cfg.IsGoogleOAuthConfigured() {
			writeJSONError(w, http.StatusServiceUnavailable, "Google Single Sign-On is not configured on this server")
			return
		}

		state, err := auth.GenerateOAuthState()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate OAuth state nonce")
			return
		}

		secure := isSecureRequest(r)
		http.SetCookie(w, auth.NewOAuthStateCookie(state, secure))

		authURL := auth.BuildGoogleAuthURL(cfg.GoogleClientID, cfg.GoogleRedirectURI, state)
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// CodeExchanger defines the function signature for exchanging an auth code with Google.
type CodeExchanger func(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*auth.GoogleTokenResponse, error)

// UserInfoFetcher defines the function signature for fetching Google user profile information.
type UserInfoFetcher func(ctx context.Context, accessToken string) (*auth.GoogleUserInfo, error)

// GoogleCallbackHandler handles the OAuth 2.0 redirect callback from Google.
// GET /api/v1/auth/google/callback
func GoogleCallbackHandler(cfg *config.Config, store *storage.UserStore) http.HandlerFunc {
	return GoogleCallbackHandlerWithServices(cfg, store, auth.ExchangeGoogleCode, auth.FetchGoogleUserInfo)
}

// GoogleCallbackHandlerWithServices allows injecting mock exchange and userinfo services for testing.
func GoogleCallbackHandlerWithServices(cfg *config.Config, store *storage.UserStore, exchange CodeExchanger, fetchUserInfo UserInfoFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if cfg == nil || !cfg.IsGoogleOAuthConfigured() {
			writeJSONError(w, http.StatusServiceUnavailable, "Google Single Sign-On is not configured on this server")
			return
		}

		query := r.URL.Query()
		if errParam := query.Get("error"); errParam != "" {
			http.SetCookie(w, auth.ClearOAuthStateCookie())
			writeJSONError(w, http.StatusBadRequest, "Google authentication was cancelled or rejected: "+errParam)
			return
		}

		stateParam := query.Get("state")
		code := query.Get("code")

		// Retrieve oauth_state cookie
		stateCookie, err := r.Cookie(auth.OAuthStateCookieName)
		// Always clear the state cookie immediately to prevent replay attacks
		http.SetCookie(w, auth.ClearOAuthStateCookie())

		if err != nil || stateCookie == nil || stateCookie.Value == "" || stateParam == "" || stateCookie.Value != stateParam {
			writeJSONError(w, http.StatusBadRequest, "Invalid or expired OAuth state parameter")
			return
		}

		if code == "" {
			writeJSONError(w, http.StatusBadRequest, "Authorization code is missing from Google response")
			return
		}

		// Exchange code for Google tokens
		tokenResp, err := exchange(r.Context(), cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI, code)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Failed to exchange authorization code with Google")
			return
		}

		// Fetch user profile from Google UserInfo endpoint
		userInfo, err := fetchUserInfo(r.Context(), tokenResp.AccessToken)
		if err != nil {
			if errors.Is(err, auth.ErrOAuthEmailNotVerified) {
				writeJSONError(w, http.StatusForbidden, "Your Google email address is unverified")
				return
			}
			writeJSONError(w, http.StatusBadRequest, "Failed to retrieve verified Google user profile")
			return
		}

		// Resolve or provision user
		var user *model.User
		user, err = store.GetUserByGoogleID(r.Context(), userInfo.Sub)
		if err != nil && !errors.Is(err, storage.ErrUserNotFound) {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query user by Google ID")
			return
		}

		if user == nil {
			// Check if existing user with matching verified email exists
			existingEmailUser, err := store.GetUserByEmail(r.Context(), userInfo.Email)
			if err != nil && !errors.Is(err, storage.ErrUserNotFound) {
				writeJSONError(w, http.StatusInternalServerError, "Failed to query user by email")
				return
			}

			if existingEmailUser != nil {
				// Link Google ID to existing account
				if err := store.LinkGoogleAccount(r.Context(), existingEmailUser.ID, userInfo.Sub); err != nil {
					writeJSONError(w, http.StatusInternalServerError, "Failed to link Google account")
					return
				}
				user = existingEmailUser
			} else {
				// Provision new Google user account
				newUser, err := store.CreateGoogleUser(r.Context(), userInfo.Email, userInfo.Name, userInfo.Sub)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, "Failed to provision user account")
					return
				}
				user = newUser
			}
		}

		// Issue authenticated session token
		sessionToken, err := auth.GenerateSecureToken(auth.DefaultTokenBytes)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate session token")
			return
		}

		expiresAt := time.Now().UTC().Add(auth.SessionDuration)
		session := &model.Session{
			Token:     sessionToken,
			UserID:    user.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
		}

		if err := store.CreateSession(r.Context(), session); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to persist user session")
			return
		}

		secure := isSecureRequest(r)
		http.SetCookie(w, auth.NewSessionCookie(sessionToken, expiresAt, secure))

		// Redirect to dashboard root
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

