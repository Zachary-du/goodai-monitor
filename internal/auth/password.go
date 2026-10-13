package auth

import (
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength defines the minimum allowed characters for a user password.
	MinPasswordLength = 8

	// BcryptCost defines the work factor for password hashing (cost >= 10).
	BcryptCost = 10
)

var (
	// ErrPasswordTooShort indicates that the candidate password is below the minimum length.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")

	// ErrInvalidPassword indicates authentication failure due to mismatched password.
	ErrInvalidPassword = errors.New("invalid email or password")

	// ErrInvalidEmail indicates that the email address is malformed.
	ErrInvalidEmail = errors.New("invalid email address format")
)

// HashPassword hashes a plaintext password using bcrypt with a minimum cost of 10.
// It returns an error if the password is shorter than MinPasswordLength characters.
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyPassword checks if the provided plaintext password matches the stored bcrypt hash.
func VerifyPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// ValidateEmail checks if the candidate string adheres to standard email format.
func ValidateEmail(email string) bool {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || len(trimmed) > 254 {
		return false
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return false
	}

	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	// Host part must contain at least one dot and non-empty domain name
	hostParts := strings.Split(parts[1], ".")
	if len(hostParts) < 2 {
		return false
	}
	for _, p := range hostParts {
		if p == "" {
			return false
		}
	}

	return true
}
