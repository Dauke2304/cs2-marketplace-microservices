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
	"github.com/patrickmn/go-cache"
)

type UserUseCase struct {
	userRepo    models.UserRepository
	sessionRepo models.SessionRepository
	tokenRepo   models.PasswordResetTokenRepository
	emailSender email.Sender
	cache       *cache.Cache
}

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrUsernameExists      = errors.New("username already exists")
	ErrWrongPassword       = errors.New("current password is incorrect")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// Cache key prefixes
const (
	CacheKeyUserByID       = "user:id:"
	CacheKeyUserByEmail    = "user:email:"
	CacheKeyUserByUsername = "user:username:"
	CacheKeySession        = "session:"
	CacheKeyBalance        = "balance:"
)

func NewUserUseCase(
	ur models.UserRepository,
	sr models.SessionRepository,
	tr models.PasswordResetTokenRepository,
	es email.Sender,
	nats *messaging.Client,
) *UserUseCase {
	// Initialize cache with 5 minute default expiration and 10 minute cleanup interval
	c := cache.New(5*time.Minute, 10*time.Minute)

	uc := &UserUseCase{
		userRepo:    ur,
		sessionRepo: sr,
		tokenRepo:   tr,
		emailSender: es,
		cache:       c,
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

// Cache helper methods
func (uc *UserUseCase) getUserFromCache(key string) (*models.User, bool) {
	if cached, found := uc.cache.Get(key); found {
		if user, ok := cached.(*models.User); ok {
			log.Printf("Cache HIT for key: %s", key)
			return user, true
		}
	}
	log.Printf("Cache MISS for key: %s", key)
	return nil, false
}

func (uc *UserUseCase) setUserCache(key string, user *models.User) {
	uc.cache.Set(key, user, cache.DefaultExpiration)
	log.Printf("Cache SET for key: %s", key)
}

func (uc *UserUseCase) invalidateUserCache(userID, email, username string) {
	uc.cache.Delete(CacheKeyUserByID + userID)
	uc.cache.Delete(CacheKeyUserByEmail + email)
	uc.cache.Delete(CacheKeyUserByUsername + username)
	uc.cache.Delete(CacheKeyBalance + userID)
	log.Printf("Cache INVALIDATED for user: %s", userID)
}

// Auth Use Cases
func (uc *UserUseCase) Register(ctx context.Context, username, email, password string) (*models.User, string, error) {
	// Check if email exists (with cache)
	if _, found := uc.getUserFromCache(CacheKeyUserByEmail + email); found {
		return nil, "", errors.New("email already exists")
	}
	if _, err := uc.userRepo.GetUserByEmail(ctx, email); err == nil {
		return nil, "", errors.New("email already exists")
	}

	// Check if username exists (with cache)
	if _, found := uc.getUserFromCache(CacheKeyUserByUsername + username); found {
		return nil, "", errors.New("username already exists")
	}
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

	// Cache the new user
	uc.setUserCache(CacheKeyUserByID+user.ID.Hex(), user)
	uc.setUserCache(CacheKeyUserByEmail+user.Email, user)
	uc.setUserCache(CacheKeyUserByUsername+user.Username, user)

	token, err := uc.createSession(ctx, user.ID.Hex())
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *UserUseCase) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	// Try cache first
	var user *models.User
	var err error

	if cachedUser, found := uc.getUserFromCache(CacheKeyUserByUsername + username); found {
		user = cachedUser
	} else {
		user, err = uc.userRepo.GetUserByUsername(ctx, username)
		if err != nil {
			return nil, "", errors.New("invalid credentials")
		}
		// Cache the user
		uc.setUserCache(CacheKeyUserByUsername+username, user)
		uc.setUserCache(CacheKeyUserByID+user.ID.Hex(), user)
		uc.setUserCache(CacheKeyUserByEmail+user.Email, user)
	}

	if !user.CheckPassword(password) {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := uc.createSession(ctx, user.ID.Hex())
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (uc *UserUseCase) Logout(ctx context.Context, token string) error {
	// Remove session from cache
	uc.cache.Delete(CacheKeySession + token)
	return uc.sessionRepo.DeleteSession(ctx, token)
}

func (uc *UserUseCase) ValidateSession(ctx context.Context, token string) (*models.User, error) {
	// Try cache first for session
	if cached, found := uc.cache.Get(CacheKeySession + token); found {
		if session, ok := cached.(*models.Session); ok {
			if session.ExpiresAt.Before(time.Now()) {
				uc.cache.Delete(CacheKeySession + token)
				_ = uc.sessionRepo.DeleteSession(ctx, token)
				return nil, errors.New("session expired")
			}

			// Get user from cache
			if user, found := uc.getUserFromCache(CacheKeyUserByID + session.UserID); found {
				return user, nil
			}
		}
	}

	// Fallback to database
	session, err := uc.sessionRepo.GetSession(ctx, token)
	if err != nil {
		return nil, errors.New("invalid session")
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = uc.sessionRepo.DeleteSession(ctx, token)
		uc.cache.Delete(CacheKeySession + token)
		return nil, errors.New("session expired")
	}

	// Cache the session
	uc.cache.Set(CacheKeySession+token, session, time.Until(session.ExpiresAt))

	user, err := uc.GetUserProfile(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Password Use Cases
func (uc *UserUseCase) ForgotPassword(ctx context.Context, email string) error {
	var user *models.User
	var err error

	// Try cache first
	if cachedUser, found := uc.getUserFromCache(CacheKeyUserByEmail + email); found {
		user = cachedUser
	} else {
		user, err = uc.userRepo.GetUserByEmail(ctx, email)
		if err != nil {
			return nil // Don't reveal if user doesn't exist
		}
		uc.setUserCache(CacheKeyUserByEmail+email, user)
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

	var user *models.User
	if cachedUser, found := uc.getUserFromCache(CacheKeyUserByEmail + resetToken.Email); found {
		user = cachedUser
	} else {
		user, err = uc.userRepo.GetUserByEmail(ctx, resetToken.Email)
		if err != nil {
			return errors.New("user not found")
		}
	}

	hashedPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate cache for this user
	uc.invalidateUserCache(user.ID.Hex(), user.Email, user.Username)

	// Delete all existing sessions for security
	_ = uc.sessionRepo.DeleteSessionsForUser(ctx, user.ID.Hex())
	return uc.tokenRepo.DeletePasswordResetToken(ctx, token)
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := uc.GetUserProfile(ctx, userID) // This will use cache
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
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate cache for this user
	uc.invalidateUserCache(user.ID.Hex(), user.Email, user.Username)

	return nil
}

// User Management Use Cases
func (uc *UserUseCase) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	// Try cache first
	if user, found := uc.getUserFromCache(CacheKeyUserByID + userID); found {
		return user, nil
	}

	// Fallback to database
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the user
	uc.setUserCache(CacheKeyUserByID+userID, user)
	uc.setUserCache(CacheKeyUserByEmail+user.Email, user)
	uc.setUserCache(CacheKeyUserByUsername+user.Username, user)

	return user, nil
}

func (uc *UserUseCase) UpdateUserProfile(ctx context.Context, userID, username, email string) (*models.User, error) {
	user, err := uc.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldEmail := user.Email
	oldUsername := user.Username

	// Check if new email is available
	if email != user.Email {
		if _, found := uc.getUserFromCache(CacheKeyUserByEmail + email); found {
			return nil, errors.New("email already in use")
		}
		if _, err := uc.userRepo.GetUserByEmail(ctx, email); err == nil {
			return nil, errors.New("email already in use")
		}
	}

	// Check if new username is available
	if username != user.Username {
		if _, found := uc.getUserFromCache(CacheKeyUserByUsername + username); found {
			return nil, errors.New("username already in use")
		}
		if _, err := uc.userRepo.GetUserByUsername(ctx, username); err == nil {
			return nil, errors.New("username already in use")
		}
	}

	user.Username = username
	user.Email = email
	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate old cache entries
	uc.cache.Delete(CacheKeyUserByEmail + oldEmail)
	uc.cache.Delete(CacheKeyUserByUsername + oldUsername)

	// Set new cache entries
	uc.setUserCache(CacheKeyUserByID+userID, user)
	uc.setUserCache(CacheKeyUserByEmail+email, user)
	uc.setUserCache(CacheKeyUserByUsername+username, user)

	return user, nil
}

// Balance Use Cases
func (uc *UserUseCase) GetBalance(ctx context.Context, userID string) (float64, error) {
	// Try cache first
	if cached, found := uc.cache.Get(CacheKeyBalance + userID); found {
		if balance, ok := cached.(float64); ok {
			log.Printf("Balance cache HIT for user: %s", userID)
			return balance, nil
		}
	}

	user, err := uc.GetUserProfile(ctx, userID) // This will use user cache
	if err != nil {
		return 0, err
	}

	// Cache the balance for 1 minute (shorter than user cache since balance changes more frequently)
	uc.cache.Set(CacheKeyBalance+userID, user.Balance, 1*time.Minute)
	log.Printf("Balance cache SET for user: %s", userID)

	return user.Balance, nil
}

func (uc *UserUseCase) UpdateBalance(ctx context.Context, userID string, amount float64) error {
	err := uc.userRepo.UpdateUserBalance(ctx, userID, amount)
	if err != nil {
		return err
	}

	// Invalidate balance and user cache
	uc.cache.Delete(CacheKeyBalance + userID)
	uc.cache.Delete(CacheKeyUserByID + userID)

	return nil
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, userID string) error {
	// Get user first to get email/username for cache invalidation
	user, err := uc.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}

	err = uc.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}

	// Invalidate all cache entries for this user
	uc.invalidateUserCache(userID, user.Email, user.Username)

	return nil
}

func (uc *UserUseCase) TransferBalance(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	// Verify sender has sufficient balance
	fromUser, err := uc.GetUserProfile(ctx, fromUserID)
	if err != nil {
		return errors.New("sender not found")
	}

	if fromUser.Balance < amount {
		return errors.New("insufficient balance")
	}

	// Verify recipient exists
	if _, err := uc.GetUserProfile(ctx, toUserID); err != nil {
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

	// Invalidate balance cache for both users
	uc.cache.Delete(CacheKeyBalance + fromUserID)
	uc.cache.Delete(CacheKeyBalance + toUserID)
	uc.cache.Delete(CacheKeyUserByID + fromUserID)
	uc.cache.Delete(CacheKeyUserByID + toUserID)

	return nil
}

// Admin Use Cases
func (uc *UserUseCase) AdminGetAllUsers(ctx context.Context, page, limit int64) ([]*models.User, error) {
	// Admin operations typically don't use cache as they need fresh data
	return uc.userRepo.GetAllUsers(ctx, page, limit)
}

func (uc *UserUseCase) AdminUpdateUser(ctx context.Context, adminID, userID string, updates *models.User) (*models.User, error) {
	// Verify admin privileges
	admin, err := uc.GetUserProfile(ctx, adminID)
	if err != nil || !admin.IsAdmin {
		return nil, errors.New("unauthorized")
	}

	user, err := uc.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldEmail := user.Email
	oldUsername := user.Username

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

	// Invalidate old cache entries if email/username changed
	if oldEmail != user.Email {
		uc.cache.Delete(CacheKeyUserByEmail + oldEmail)
	}
	if oldUsername != user.Username {
		uc.cache.Delete(CacheKeyUserByUsername + oldUsername)
	}

	// Invalidate user cache
	uc.invalidateUserCache(userID, user.Email, user.Username)

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

	// Cache the session
	uc.cache.Set(CacheKeySession+token, session, 24*time.Hour)

	return token, nil
}
