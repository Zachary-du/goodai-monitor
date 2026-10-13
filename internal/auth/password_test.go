package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_Success(t *testing.T) {
	password := "SecretPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("expected non-empty hash")
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("failed to retrieve cost: %v", err)
	}
	if cost < BcryptCost {
		t.Fatalf("expected bcrypt cost >= %d, got %d", BcryptCost, cost)
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	shortPasswords := []string{"", "123", "short7"}
	for _, pwd := range shortPasswords {
		_, err := HashPassword(pwd)
		if err != ErrPasswordTooShort {
			t.Errorf("expected ErrPasswordTooShort for '%s', got %v", pwd, err)
		}
	}
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "CorrectHorseBatteryStaple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if err := VerifyPassword(hash, password); err != nil {
		t.Fatalf("expected successful verification, got %v", err)
	}
}

func TestVerifyPassword_Mismatch(t *testing.T) {
	password := "CorrectHorseBatteryStaple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if err := VerifyPassword(hash, "WrongPassword456"); err != ErrInvalidPassword {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"admin@goodai.co.nz", true},
		{"first.last+tag@sub.domain.org", true},
		{"", false},
		{"plainaddress", false},
		{"@missingusername.com", false},
		{"username@.com", false},
		{"username@domain", false},
		{"user name@example.com", false},
	}

	for _, tt := range tests {
		result := ValidateEmail(tt.email)
		if result != tt.valid {
			t.Errorf("ValidateEmail(%q) = %v, expected %v", tt.email, result, tt.valid)
		}
	}
}
