package mongo

import (
	"context"
	"cs2-marketplace-microservices/transaction-service/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TransactionRepository struct {
	collection *mongo.Collection
}

func NewTransactionRepository(db *mongo.Database) *TransactionRepository {
	return &TransactionRepository{
		collection: db.Collection("transactions"),
	}
}

// CreateTransaction inserts a new transaction into the database
func (r *TransactionRepository) CreateTransaction(ctx context.Context, transaction *models.Transaction) (*models.Transaction, error) {
	now := time.Now()
	transaction.CreatedAt = now
	transaction.UpdatedAt = now
	transaction.Date = now.Format("2006-01-02 15:04:05")

	result, err := r.collection.InsertOne(ctx, transaction)
	if err != nil {
		return nil, err
	}

	transaction.ID = result.InsertedID.(primitive.ObjectID)
	return transaction, nil
}

// GetTransactionByID retrieves a transaction by its ID
func (r *TransactionRepository) GetTransactionByID(ctx context.Context, id primitive.ObjectID) (*models.Transaction, error) {
	var transaction models.Transaction
	filter := bson.M{"_id": id}

	err := r.collection.FindOne(ctx, filter).Decode(&transaction)
	if err != nil {
		return nil, err
	}

	return &transaction, nil
}

// UpdateTransaction updates a transaction
func (r *TransactionRepository) UpdateTransaction(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.Transaction, error) {
	update["updated_at"] = time.Now()

	filter := bson.M{"_id": id}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedTransaction models.Transaction
	err := r.collection.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, opts).Decode(&updatedTransaction)
	if err != nil {
		return nil, err
	}

	return &updatedTransaction, nil
}

// DeleteTransaction removes a transaction from the database
func (r *TransactionRepository) DeleteTransaction(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// GetTransactionsByUserID retrieves all transactions for a specific user (buyer or seller)
func (r *TransactionRepository) GetTransactionsByUserID(ctx context.Context, userID primitive.ObjectID, status models.TransactionStatus, txType models.TransactionType, limit, offset int32) ([]models.Transaction, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"buyer_id": userID},
			{"seller_id": userID},
		},
	}

	// Add optional filters
	if status != "" {
		filter["status"] = status
	}
	if txType != "" {
		filter["type"] = txType
	}

	// Count total documents
	totalCount, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Set up options for pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by newest first

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var transactions []models.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, 0, err
	}

	return transactions, totalCount, nil
}

// GetTransactionsBySkinID retrieves all transactions for a specific skin
func (r *TransactionRepository) GetTransactionsBySkinID(ctx context.Context, skinID primitive.ObjectID) ([]models.Transaction, error) {
	filter := bson.M{"skin_id": skinID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []models.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

// GetTransactionsByStatus retrieves transactions filtered by status
func (r *TransactionRepository) GetTransactionsByStatus(ctx context.Context, status models.TransactionStatus, limit, offset int32) ([]models.Transaction, int64, error) {
	filter := bson.M{"status": status}

	// Count total documents
	totalCount, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var transactions []models.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, 0, err
	}

	return transactions, totalCount, nil
}

// GetAllTransactions retrieves all transactions with pagination
func (r *TransactionRepository) GetAllTransactions(ctx context.Context, limit, offset int32) ([]models.Transaction, int64, error) {
	filter := bson.M{}

	// Count total documents
	totalCount, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var transactions []models.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, 0, err
	}

	return transactions, totalCount, nil
}

// GetTransactionStats calculates transaction statistics
func (r *TransactionRepository) GetTransactionStats(ctx context.Context, userID *primitive.ObjectID, startDate, endDate string) (*TransactionStats, error) {
	filter := bson.M{}

	// Add user filter if provided
	if userID != nil {
		filter["$or"] = []bson.M{
			{"buyer_id": *userID},
			{"seller_id": *userID},
		}
	}

	// Add date range filter if provided
	if startDate != "" || endDate != "" {
		dateFilter := bson.M{}
		if startDate != "" {
			dateFilter["$gte"] = startDate
		}
		if endDate != "" {
			dateFilter["$lte"] = endDate
		}
		filter["date"] = dateFilter
	}

	// Aggregation pipeline for statistics
	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":                nil,
				"total_transactions": bson.M{"$sum": 1},
				"total_amount":       bson.M{"$sum": "$amount"},
				"successful_transactions": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", "COMPLETED"}},
							1,
							0,
						},
					},
				},
				"failed_transactions": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$in": []interface{}{"$status", []string{"FAILED", "CANCELLED"}}},
							1,
							0,
						},
					},
				},
				"average_amount": bson.M{"$avg": "$amount"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		// Return empty stats if no transactions found
		return &TransactionStats{}, nil
	}

	result := results[0]
	stats := &TransactionStats{
		TotalTransactions:      getInt32FromBSON(result, "total_transactions"),
		TotalAmount:            getFloat64FromBSON(result, "total_amount"),
		SuccessfulTransactions: getInt32FromBSON(result, "successful_transactions"),
		FailedTransactions:     getInt32FromBSON(result, "failed_transactions"),
		AverageAmount:          getFloat64FromBSON(result, "average_amount"),
	}

	return stats, nil
}

// TransactionStats represents aggregated transaction statistics
type TransactionStats struct {
	TotalTransactions      int32   `json:"total_transactions"`
	TotalAmount            float64 `json:"total_amount"`
	SuccessfulTransactions int32   `json:"successful_transactions"`
	FailedTransactions     int32   `json:"failed_transactions"`
	AverageAmount          float64 `json:"average_amount"`
}

// Helper functions to safely extract values from BSON
func getInt32FromBSON(data bson.M, key string) int32 {
	if val, ok := data[key]; ok {
		if intVal, ok := val.(int32); ok {
			return intVal
		}
		if intVal, ok := val.(int); ok {
			return int32(intVal)
		}
	}
	return 0
}

func getFloat64FromBSON(data bson.M, key string) float64 {
	if val, ok := data[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
		if intVal, ok := val.(int); ok {
			return float64(intVal)
		}
		if intVal, ok := val.(int32); ok {
			return float64(intVal)
		}
	}
	return 0.0
}
