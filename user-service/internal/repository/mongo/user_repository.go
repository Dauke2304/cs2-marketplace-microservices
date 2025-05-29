package mongo

import (
	"context"
	"cs2-marketplace-microservices/user-service/internal/models"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) models.UserRepository {
	return &userRepository{
		collection: db.Collection("users"),
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	return err
}
func (r *userRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *models.User) error {
	if user.ID.IsZero() {
		return errors.New("user ID is required for update")
	}

	_, err := r.collection.UpdateByID(ctx, user.ID, bson.M{"$set": user})
	return err
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// UpdateUserBalance changes a user’s balance.
func (r *userRepository) UpdateUserBalance(ctx context.Context, id string, amount float64) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$inc": bson.M{"balance": amount}},
	)
	return err
}

// GetAllUsers retrieves users with pagination.
func (r *userRepository) GetAllUsers(ctx context.Context, page, limit int64) ([]*models.User, error) {
	opts := options.Find().
		SetSkip((page - 1) * limit).
		SetLimit(limit).
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			continue
		}
		users = append(users, &user)
	}

	return users, nil
}

// GetUserCount returns the total number of users.
func (r *userRepository) GetUserCount(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

type sessionRepository struct {
	collection *mongo.Collection
}

func NewSessionRepository(db *mongo.Database) *sessionRepository {
	return &sessionRepository{
		collection: db.Collection("sessions"),
	}
}

func (r *sessionRepository) CreateSession(ctx context.Context, session *models.Session) error {
	_, err := r.collection.InsertOne(ctx, session)
	return err
}

func (r *sessionRepository) GetSession(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"token": token})
	return err
}

func (r *sessionRepository) DeleteSessionsForUser(ctx context.Context, userID string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}

// PasswordResetTokenRepository implementation
type passwordResetTokenRepository struct {
	collection *mongo.Collection
}

func NewPasswordResetTokenRepository(db *mongo.Database) *passwordResetTokenRepository {
	return &passwordResetTokenRepository{
		collection: db.Collection("password_reset_tokens"),
	}
}

func (r *passwordResetTokenRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	_, err := r.collection.InsertOne(ctx, token)
	return err
}

func (r *passwordResetTokenRepository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&resetToken)
	if err != nil {
		return nil, err
	}
	return &resetToken, nil
}

func (r *passwordResetTokenRepository) DeletePasswordResetToken(ctx context.Context, token string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"token": token})
	return err
}

func (r *passwordResetTokenRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	now := time.Now()
	_, err := r.collection.DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": now},
	})
	return err
}
