package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"cs2-marketplace-microservices/user-service/internal/models"
	"cs2-marketplace-microservices/user-service/pkg/email"
	"cs2-marketplace-microservices/user-service/pkg/messaging"
	"cs2-marketplace-microservices/user-service/pkg/security"

	natsgo "github.com/nats-io/nats.go"
)

type UserUseCase struct {
	userRepo    models.UserRepository
	sessionRepo models.SessionRepository
	tokenRepo   models.PasswordResetTokenRepository
	emailSender email.Sender
}

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrUsernameExists      = errors.New("username already exists")
	ErrWrongPassword       = errors.New("current password is incorrect")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

func NewUserUseCase(
	ur models.UserRepository,
	sr models.SessionRepository,
	tr models.PasswordResetTokenRepository,
	es email.Sender,
	nats *messaging.Client, // Add NATS client
) *UserUseCase {
	uc := &UserUseCase{
		userRepo:    ur,
		sessionRepo: sr,
		tokenRepo:   tr,
		emailSender: es,
	}

	// Subscribe to skin.created events
	if nats != nil {
		sub, err := nats.Conn.Subscribe("skin.created", func(m *natsgo.Msg) {
			log.Printf("RECEIVED SKIN ID: %s", string(m.Data))
		})
		if err != nil {
			log.Printf("Failed to subscribe: %v", err)
		} else {
			log.Printf("Subscribed to skin.created (ID: %s)", sub.Subject)
		}
	}

	return uc
}

// Auth Use Cases
func (uc *UserUseCase) Register(ctx context.Context, username, email, password string) (*models.User, string, error) {
	// Check if email exists
	if _, err := uc.userRepo.GetUserByEmail(ctx, email); err == nil {
		return nil, "", errors.New("email already exists")
	}

	// Check if username exists
	if _, err := uc.userRepo.GetUserByUsername(ctx, username); err == nil {
		return nil, "", errors.New("username already exists")
	}

	// Create admin user if it's the first registration
	isAdmin := false
	if count, _ := uc.userRepo.GetUserCount(ctx); count == 0 {
		isAdmin = true
	}

	user, err := models.NewUser(username, email, password, isAdmin)
	if err != nil {
		return nil, "", err
	}

	if err := uc.userRepo.CreateUser(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := uc.createSession(ctx, user.ID.Hex())
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *UserUseCase) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	user, err := uc.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		fmt.Println("d1")
		return nil, "", errors.New("invalid credentials")
	}

	if !user.CheckPassword(password) {
		fmt.Println("d2")
		fmt.Println(password)

		return nil, "", errors.New("invalid credentials")
	}

	token, err := uc.createSession(ctx, user.ID.Hex())
	if err != nil {
		return nil, "", err
	}
	fmt.Println("sessiontokenUC:", token)

	return user, token, nil
}

func (uc *UserUseCase) Logout(ctx context.Context, token string) error {
	return uc.sessionRepo.DeleteSession(ctx, token)
}

func (uc *UserUseCase) ValidateSession(ctx context.Context, token string) (*models.User, error) {
	session, err := uc.sessionRepo.GetSession(ctx, token)
	fmt.Println(token)
	if err != nil {
		fmt.Println("a1")
		return nil, errors.New("invalid session")
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = uc.sessionRepo.DeleteSession(ctx, token)
		fmt.Println("a2")
		return nil, errors.New("session expired")
	}

	return uc.userRepo.GetUserByID(ctx, session.UserID)
}

// Password Use Cases
func (uc *UserUseCase) ForgotPassword(ctx context.Context, email string) error {
	user, err := uc.userRepo.GetUserByEmail(ctx, email)

	if err != nil {
		return nil // Don't reveal if user doesn't exist
	}

	token := security.GenerateToken()
	resetToken := &models.PasswordResetToken{
		Token:     token,
		Email:     user.Email,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := uc.tokenRepo.CreatePasswordResetToken(ctx, resetToken); err != nil {
		return err
	}

	// Send email with reset link
	resetLink := fmt.Sprintf("https://yourdomain.com/reset-password?token=%s", token)
	body := fmt.Sprintf("Click this link to reset your password: %s", resetLink)
	return uc.emailSender.SendEmail(user.Email, "Password Reset Request", body)
}

func (uc *UserUseCase) ResetPassword(ctx context.Context, token, newPassword string) error {
	resetToken, err := uc.tokenRepo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	if resetToken.ExpiresAt.Before(time.Now()) {
		_ = uc.tokenRepo.DeletePasswordResetToken(ctx, token)
		return errors.New("token expired")
	}

	user, err := uc.userRepo.GetUserByEmail(ctx, resetToken.Email)
	if err != nil {
		return errors.New("user not found")
	}

	hashedPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Delete all existing sessions for security
	_ = uc.sessionRepo.DeleteSessionsForUser(ctx, user.ID.Hex())
	return uc.tokenRepo.DeletePasswordResetToken(ctx, token)
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !user.CheckPassword(currentPassword) {
		return errors.New("current password is incorrect")
	}

	hashedPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return uc.userRepo.UpdateUser(ctx, user)
}

// User Management Use Cases
func (uc *UserUseCase) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	return uc.userRepo.GetUserByID(ctx, userID)
}

func (uc *UserUseCase) UpdateUserProfile(ctx context.Context, userID, username, email string) (*models.User, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if new email is available
	if email != user.Email {
		if _, err := uc.userRepo.GetUserByEmail(ctx, email); err == nil {
			return nil, errors.New("email already in use")
		}
	}

	// Check if new username is available
	if username != user.Username {
		if _, err := uc.userRepo.GetUserByUsername(ctx, username); err == nil {
			return nil, errors.New("username already in use")
		}
	}

	user.Username = username
	user.Email = email
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Balance Use Cases
func (uc *UserUseCase) GetBalance(ctx context.Context, userID string) (float64, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

func (uc *UserUseCase) UpdateBalance(ctx context.Context, userID string, amount float64) error {
	return uc.userRepo.UpdateUserBalance(ctx, userID, amount)
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, userID string) error {
	return uc.userRepo.DeleteUser(ctx, userID)
}

func (uc *UserUseCase) TransferBalance(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	// Verify sender has sufficient balance
	fromUser, err := uc.userRepo.GetUserByID(ctx, fromUserID)
	if err != nil {
		return errors.New("sender not found")
	}

	if fromUser.Balance < amount {
		return errors.New("insufficient balance")
	}

	// Verify recipient exists
	if _, err := uc.userRepo.GetUserByID(ctx, toUserID); err != nil {
		return errors.New("recipient not found")
	}

	// Perform transfer
	if err := uc.userRepo.UpdateUserBalance(ctx, fromUserID, -amount); err != nil {
		return err
	}

	if err := uc.userRepo.UpdateUserBalance(ctx, toUserID, amount); err != nil {
		// Attempt to rollback
		_ = uc.userRepo.UpdateUserBalance(ctx, fromUserID, amount)
		return err
	}

	return nil
}

// Admin Use Cases
func (uc *UserUseCase) AdminGetAllUsers(ctx context.Context, page, limit int64) ([]*models.User, error) {
	return uc.userRepo.GetAllUsers(ctx, page, limit)
}

func (uc *UserUseCase) AdminUpdateUser(ctx context.Context, adminID, userID string, updates *models.User) (*models.User, error) {
	// Verify admin privileges
	admin, err := uc.userRepo.GetUserByID(ctx, adminID)
	if err != nil || !admin.IsAdmin {
		return nil, errors.New("unauthorized")
	}

	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if updates.Username != "" {
		user.Username = updates.Username
	}
	if updates.Email != "" {
		user.Email = updates.Email
	}
	if updates.Balance != 0 {
		user.Balance = updates.Balance
	}
	if updates.IsAdmin {
		user.IsAdmin = updates.IsAdmin
	}

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Helper methods
func (uc *UserUseCase) createSession(ctx context.Context, userID string) (string, error) {
	token := security.GenerateToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	session := &models.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	if err := uc.sessionRepo.CreateSession(ctx, session); err != nil {
		return "", err
	}

	return token, nil
}
