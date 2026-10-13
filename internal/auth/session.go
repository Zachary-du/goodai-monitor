package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"
)

const (
	// SessionCookieName defines the standard name of the session cookie.
	SessionCookieName = "goodai_session"

	// SessionDuration defines the validity period of a user session (14 days).
	SessionDuration = 14 * 24 * time.Hour

	// DefaultTokenBytes defines the default entropy (32 bytes = 256 bits = 64 hex chars).
	DefaultTokenBytes = 32
)

var (
	// ErrFailedToGenerateToken indicates an error reading from the cryptographic PRNG.
	ErrFailedToGenerateToken = errors.New("failed to generate secure random token")
)

// GenerateSecureToken creates a cryptographically secure random hexadecimal string.
// If byteLength <= 0, it defaults to DefaultTokenBytes (32 bytes).
func GenerateSecureToken(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = DefaultTokenBytes
	}

	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", ErrFailedToGenerateToken
	}

	return hex.EncodeToString(buf), nil
}

// NewSessionCookie constructs an HttpOnly, SameSite=Lax cookie containing the session token.
func NewSessionCookie(token string, expiresAt time.Time, secure bool) *http.Cookie {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearSessionCookie returns an expired cookie that instructs the client browser to delete the session token.
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
