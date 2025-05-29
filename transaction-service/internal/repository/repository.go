package repository

import (
	"context"
	"cs2-marketplace-microservices/transaction-service/internal/models"
	"cs2-marketplace-microservices/transaction-service/internal/repository/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, transaction *models.Transaction) (*models.Transaction, error)
	GetTransactionByID(ctx context.Context, id primitive.ObjectID) (*models.Transaction, error)
	UpdateTransaction(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.Transaction, error)
	DeleteTransaction(ctx context.Context, id primitive.ObjectID) error
	GetTransactionsByUserID(ctx context.Context, userID primitive.ObjectID, status models.TransactionStatus, txType models.TransactionType, limit, offset int32) ([]models.Transaction, int64, error)
	GetTransactionsBySkinID(ctx context.Context, skinID primitive.ObjectID) ([]models.Transaction, error)
	GetTransactionsByStatus(ctx context.Context, status models.TransactionStatus, limit, offset int32) ([]models.Transaction, int64, error)
	GetAllTransactions(ctx context.Context, limit, offset int32) ([]models.Transaction, int64, error)
	GetTransactionStats(ctx context.Context, userID *primitive.ObjectID, startDate, endDate string) (*mongo.TransactionStats, error)
}

type Repositories struct {
	Transaction TransactionRepository
}

func NewRepositories(transactionRepo TransactionRepository) *Repositories {
	return &Repositories{
		Transaction: transactionRepo,
	}
}
