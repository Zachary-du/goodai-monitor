package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

type contextKey string

const (
	// UserIDContextKey is the context key for the authenticated user's ID.
	UserIDContextKey contextKey = "goodai_user_id"

	// UserContextKey is the context key for the authenticated user model.
	UserContextKey contextKey = "goodai_user"
)

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	val, ok := ctx.Value(UserIDContextKey).(int64)
	return val, ok
}

// UserFromContext extracts the authenticated User struct from context.
func UserFromContext(ctx context.Context) (*model.User, bool) {
	val, ok := ctx.Value(UserContextKey).(*model.User)
	return val, ok
}

// AuthMiddleware inspects incoming requests for a valid session cookie.
// If valid, it injects the verified userID and User into the request context.
// Public endpoints pass through without hindrance.
func AuthMiddleware(store *storage.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			session, err := store.GetSessionByToken(r.Context(), cookie.Value)
			if err != nil {
				// Expired or invalid session cookie; continue without user context
				next.ServeHTTP(w, r)
				return
			}

			user, err := store.GetUserByID(r.Context(), session.UserID)
			if err != nil {
				// User record not found; continue without user context
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, user.ID)
			ctx = context.WithValue(ctx, UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth wraps an http.HandlerFunc, returning 401 Unauthorized if the user is not authenticated.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Authentication required",
			})
			return
		}
		next(w, r)
	}
}

// RequireAuthHandler wraps an http.Handler, returning 401 Unauthorized if the user is not authenticated.
func RequireAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Authentication required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
