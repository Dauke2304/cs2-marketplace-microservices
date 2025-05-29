package models

import (
	"context"
)

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	// Basic CRUD operations
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error

	// Balance operations
	UpdateUserBalance(ctx context.Context, id string, amount float64) error

	// Admin operations
	GetAllUsers(ctx context.Context, page, limit int64) ([]*User, error)
	GetUserCount(ctx context.Context) (int64, error)
}

// SessionRepository defines the interface for session management
type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteSessionsForUser(ctx context.Context, userID string) error
}

// PasswordResetTokenRepository defines the interface for password reset tokens
type PasswordResetTokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error)
	DeletePasswordResetToken(ctx context.Context, token string) error
	DeleteExpiredPasswordResetTokens(ctx context.Context) error
}
