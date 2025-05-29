package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a hashed password with a plaintext password
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateToken creates a secure random token
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// TokenManager handles token generation and validation
type TokenManager struct {
	secretKey     string
	tokenDuration time.Duration
}

func NewTokenManager(secretKey string, duration time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:     secretKey,
		tokenDuration: duration,
	}
}

// GenerateToken generates a new JWT token
func (tm *TokenManager) GenerateToken(userID string) (string, error) {
	// In a real implementation, this would generate a JWT
	// For simplicity, we'll combine userID with a random string
	token := userID + ":" + GenerateToken()
	return token, nil
}

// ValidateToken validates a token and returns the user ID
func (tm *TokenManager) ValidateToken(token string) (string, error) {
	// In a real implementation, this would validate a JWT
	// For now, we'll just split the simple token format
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return "", errors.New("invalid token format")
	}
	return parts[0], nil
}

// PasswordValidator enforces password policies
type PasswordValidator struct {
	minLength      int
	requireUpper   bool
	requireLower   bool
	requireNumber  bool
	requireSpecial bool
}

func NewPasswordValidator(minLength int, requireUpper, requireLower, requireNumber, requireSpecial bool) *PasswordValidator {
	return &PasswordValidator{
		minLength:      minLength,
		requireUpper:   requireUpper,
		requireLower:   requireLower,
		requireNumber:  requireNumber,
		requireSpecial: requireSpecial,
	}
}

func (pv *PasswordValidator) Validate(password string) error {
	if len(password) < pv.minLength {
		return fmt.Errorf("password must be at least %d characters long", pv.minLength)
	}

	if pv.requireUpper {
		if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return errors.New("password must contain at least one uppercase letter")
		}
	}

	if pv.requireLower {
		if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
			return errors.New("password must contain at least one lowercase letter")
		}
	}

	if pv.requireNumber {
		if !strings.ContainsAny(password, "0123456789") {
			return errors.New("password must contain at least one number")
		}
	}

	if pv.requireSpecial {
		if !strings.ContainsAny(password, "!@#$%^&*()-_=+[]{};:,.<>/?") {
			return errors.New("password must contain at least one special character")
		}
	}

	return nil
}
