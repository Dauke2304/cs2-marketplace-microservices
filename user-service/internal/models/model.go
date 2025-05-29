package models

import (
	"time"

	"cs2-marketplace-microservices/user-service/proto/user"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// User represents the core user entity in the system
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username" validate:"required,min=3,max=50"`
	Email     string             `bson:"email" json:"email" validate:"required,email"`
	Password  string             `bson:"password" json:"-" validate:"required,min=8"`
	Balance   float64            `bson:"balance" json:"balance"`
	IsAdmin   bool               `bson:"is_admin" json:"is_admin"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Session represents an active user session
type Session struct {
	Token     string    `bson:"token" json:"token"`
	UserID    string    `bson:"user_id" json:"user_id"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
}

// PasswordResetToken represents a password reset request
type PasswordResetToken struct {
	Token     string    `bson:"token" json:"token"`
	Email     string    `bson:"email" json:"email"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
}

// NewUser creates a new User instance with hashed password
func NewUser(username, email, password string, isAdmin bool) (*User, error) {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		Username:  username,
		Email:     email,
		Password:  hashedPassword,
		Balance:   0.0,
		IsAdmin:   isAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// HashPassword hashes the user's password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// ToProto converts User model to gRPC proto User
func (u *User) ToProto() *user.User {
	return &user.User{
		Id:        u.ID.Hex(),
		Username:  u.Username,
		Email:     u.Email,
		Balance:   u.Balance,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

// FromProto creates a User model from gRPC proto User
func FromProto(protoUser *user.User) (*User, error) {
	id, err := primitive.ObjectIDFromHex(protoUser.GetId())
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, protoUser.GetCreatedAt())
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, protoUser.GetUpdatedAt())
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        id,
		Username:  protoUser.GetUsername(),
		Email:     protoUser.GetEmail(),
		Balance:   protoUser.GetBalance(),
		IsAdmin:   protoUser.GetIsAdmin(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
