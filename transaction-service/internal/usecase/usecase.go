package usecase

import (
	"context"
	"cs2-marketplace-microservices/transaction-service/internal/models"
	"cs2-marketplace-microservices/transaction-service/internal/repository"
	"cs2-marketplace-microservices/transaction-service/proto/transaction"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionUsecase interface {
	CreateTransaction(ctx context.Context, req *transaction.CreateTransactionRequest) (*transaction.TransactionResponse, error)
	GetTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.TransactionResponse, error)
	UpdateTransaction(ctx context.Context, req *transaction.UpdateTransactionRequest) (*transaction.TransactionResponse, error)
	DeleteTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.DeleteResponse, error)
	ListTransactions(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error)
	GetTransactionsByUser(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error)
	GetTransactionsBySkin(ctx context.Context, req *transaction.GetTransactionsBySkinRequest) (*transaction.TransactionListResponse, error)
	GetTransactionsByStatus(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error)
	ProcessPurchase(ctx context.Context, req *transaction.ProcessPurchaseRequest) (*transaction.TransactionResponse, error)
	CancelTransaction(ctx context.Context, req *transaction.CancelTransactionRequest) (*transaction.TransactionResponse, error)
	GetTransactionStats(ctx context.Context, req *transaction.GetTransactionStatsRequest) (*transaction.TransactionStatsResponse, error)
	GetAllTransactions(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error)
}

type transactionUsecase struct {
	transactionRepo repository.TransactionRepository
	cache           *cache.Cache
}

// Cache key constants
const (
	transactionCachePrefix        = "transaction:"
	userTransactionsCachePrefix   = "user_transactions:"
	skinTransactionsCachePrefix   = "skin_transactions:"
	statusTransactionsCachePrefix = "status_transactions:"
	statsCachePrefix              = "stats:"
	allTransactionsCachePrefix    = "all_transactions:"
)

// Cache TTL settings
const (
	transactionCacheTTL = 5 * time.Minute  // Individual transactions
	listCacheTTL        = 2 * time.Minute  // Lists (shorter TTL as they change more often)
	statsCacheTTL       = 10 * time.Minute // Stats (longer TTL as they're expensive to compute)
)

func NewTransactionUsecase(transactionRepo repository.TransactionRepository) TransactionUsecase {
	// Create cache with default expiration of 5 minutes and cleanup every 10 minutes
	c := cache.New(5*time.Minute, 10*time.Minute)

	return &transactionUsecase{
		transactionRepo: transactionRepo,
		cache:           c,
	}
}

// Helper function to generate cache keys
func (uc *transactionUsecase) getCacheKey(prefix, id string) string {
	return fmt.Sprintf("%s%s", prefix, id)
}

// Helper function to generate list cache keys with parameters
func (uc *transactionUsecase) getListCacheKey(prefix string, params ...interface{}) string {
	key := prefix
	for _, param := range params {
		key += fmt.Sprintf("_%v", param)
	}
	return key
}

// Helper function to invalidate related caches when transaction is modified
func (uc *transactionUsecase) invalidateTransactionCaches(transactionID string, userID, skinID *primitive.ObjectID) {
	// Invalidate specific transaction cache
	uc.cache.Delete(uc.getCacheKey(transactionCachePrefix, transactionID))

	// Invalidate user-related caches if userID is provided
	if userID != nil {
		uc.cache.DeleteExpired() // Clean up to find user-related keys
		// In a real implementation, you might want to track user-related keys separately
		// For simplicity, we'll use a pattern-based approach or clear related patterns
	}

	// Invalidate skin-related caches
	if skinID != nil {
		skinKey := uc.getCacheKey(skinTransactionsCachePrefix, skinID.Hex())
		uc.cache.Delete(skinKey)
	}

	// Clear all transactions cache as it might be affected
	uc.cache.Delete(allTransactionsCachePrefix)

	// Clear stats cache as transaction changes affect statistics
	uc.clearStatsCaches()
}

func (uc *transactionUsecase) clearStatsCaches() {
	// Clear all stats-related cache entries
	items := uc.cache.Items()
	for key := range items {
		if len(key) >= len(statsCachePrefix) && key[:len(statsCachePrefix)] == statsCachePrefix {
			uc.cache.Delete(key)
		}
	}
}

func (uc *transactionUsecase) CreateTransaction(ctx context.Context, req *transaction.CreateTransactionRequest) (*transaction.TransactionResponse, error) {
	if req.GetBuyerId() == "" || req.GetSkinId() == "" || req.GetAmount() <= 0 {
		return nil, errors.New("invalid request: buyer_id, skin_id and amount are required")
	}

	buyerID, err := primitive.ObjectIDFromHex(req.GetBuyerId())
	if err != nil {
		return nil, fmt.Errorf("invalid buyer_id: %v", err)
	}

	var sellerID primitive.ObjectID
	if req.GetSellerId() != "" {
		sellerID, err = primitive.ObjectIDFromHex(req.GetSellerId())
		if err != nil {
			return nil, fmt.Errorf("invalid seller_id: %v", err)
		}
	}

	skinID, err := primitive.ObjectIDFromHex(req.GetSkinId())
	if err != nil {
		return nil, fmt.Errorf("invalid skin_id: %v", err)
	}

	newTransaction := &models.Transaction{
		BuyerID:     buyerID,
		SellerID:    sellerID,
		SkinID:      skinID,
		Amount:      req.GetAmount(),
		Status:      models.StatusPending,
		Type:        models.TypeFromProto(req.GetType()),
		Description: req.GetDescription(),
	}

	createdTransaction, err := uc.transactionRepo.CreateTransaction(ctx, newTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// Invalidate related caches after creating new transaction
	uc.invalidateTransactionCaches(createdTransaction.ID.Hex(), &buyerID, &skinID)

	return &transaction.TransactionResponse{
		Transaction: createdTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) GetTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.TransactionResponse, error) {
	if req.GetId() == "" {
		return nil, errors.New("transaction id is required")
	}

	// Try to get from cache first
	cacheKey := uc.getCacheKey(transactionCachePrefix, req.GetId())
	if cached, found := uc.cache.Get(cacheKey); found {
		if trans, ok := cached.(*models.Transaction); ok {
			return &transaction.TransactionResponse{
				Transaction: trans.ToProto(),
			}, nil
		}
	}

	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid transaction id: %v", err)
	}

	trans, err := uc.transactionRepo.GetTransactionByID(ctx, objID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	// Cache the result
	uc.cache.Set(cacheKey, trans, transactionCacheTTL)

	return &transaction.TransactionResponse{
		Transaction: trans.ToProto(),
	}, nil
}

func (uc *transactionUsecase) UpdateTransaction(ctx context.Context, req *transaction.UpdateTransactionRequest) (*transaction.TransactionResponse, error) {
	if req.GetId() == "" {
		return nil, errors.New("transaction id is required")
	}

	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid transaction id: %v", err)
	}

	update := bson.M{}
	if req.Status != transaction.TransactionStatus_PENDING {
		update["status"] = models.StatusFromProto(req.GetStatus())
	}
	if req.GetDescription() != "" {
		update["description"] = req.GetDescription()
	}

	if len(update) == 0 {
		return nil, errors.New("no fields to update")
	}

	updatedTransaction, err := uc.transactionRepo.UpdateTransaction(ctx, objID, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %v", err)
	}

	// Invalidate caches after update
	uc.invalidateTransactionCaches(req.GetId(), &updatedTransaction.BuyerID, &updatedTransaction.SkinID)

	return &transaction.TransactionResponse{
		Transaction: updatedTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) DeleteTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.DeleteResponse, error) {
	if req.GetId() == "" {
		return nil, errors.New("transaction id is required")
	}

	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid transaction id: %v", err)
	}

	// Get transaction before deletion to invalidate related caches
	trans, err := uc.transactionRepo.GetTransactionByID(ctx, objID)
	if err != nil {
		return &transaction.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("failed to find transaction: %v", err),
		}, nil
	}

	err = uc.transactionRepo.DeleteTransaction(ctx, objID)
	if err != nil {
		return &transaction.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("failed to delete transaction: %v", err),
		}, nil
	}

	// Invalidate caches after deletion
	uc.invalidateTransactionCaches(req.GetId(), &trans.BuyerID, &trans.SkinID)

	return &transaction.DeleteResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	}, nil
}

func (uc *transactionUsecase) ListTransactions(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error) {
	return uc.GetTransactionsByUser(ctx, req)
}

func (uc *transactionUsecase) GetTransactionsByUser(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error) {
	if req.GetUserId() == "" {
		return nil, errors.New("user id is required")
	}

	// Generate cache key with all parameters
	cacheKey := uc.getListCacheKey(userTransactionsCachePrefix, req.GetUserId(), req.GetStatus(), req.GetType(), req.GetLimit(), req.GetOffset())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if response, ok := cached.(*transaction.TransactionListResponse); ok {
			return response, nil
		}
	}

	userID, err := primitive.ObjectIDFromHex(req.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %v", err)
	}

	status := models.StatusFromProto(req.GetStatus())
	txType := models.TypeFromProto(req.GetType())

	transactions, totalCount, err := uc.transactionRepo.GetTransactionsByUserID(ctx, userID, status, txType, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	response := &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}

	// Cache the result
	uc.cache.Set(cacheKey, response, listCacheTTL)

	return response, nil
}

func (uc *transactionUsecase) GetTransactionsBySkin(ctx context.Context, req *transaction.GetTransactionsBySkinRequest) (*transaction.TransactionListResponse, error) {
	if req.GetSkinId() == "" {
		return nil, errors.New("skin id is required")
	}

	// Try to get from cache first
	cacheKey := uc.getCacheKey(skinTransactionsCachePrefix, req.GetSkinId())
	if cached, found := uc.cache.Get(cacheKey); found {
		if response, ok := cached.(*transaction.TransactionListResponse); ok {
			return response, nil
		}
	}

	skinID, err := primitive.ObjectIDFromHex(req.GetSkinId())
	if err != nil {
		return nil, fmt.Errorf("invalid skin id: %v", err)
	}

	transactions, err := uc.transactionRepo.GetTransactionsBySkinID(ctx, skinID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	response := &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(len(transactions)),
	}

	// Cache the result
	uc.cache.Set(cacheKey, response, listCacheTTL)

	return response, nil
}

func (uc *transactionUsecase) GetTransactionsByStatus(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	// Generate cache key with parameters
	cacheKey := uc.getListCacheKey(statusTransactionsCachePrefix, req.GetStatus(), req.GetLimit(), req.GetOffset())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if response, ok := cached.(*transaction.TransactionListResponse); ok {
			return response, nil
		}
	}

	status := models.StatusFromProto(req.GetStatus())

	transactions, totalCount, err := uc.transactionRepo.GetTransactionsByStatus(ctx, status, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	response := &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}

	// Cache the result
	uc.cache.Set(cacheKey, response, listCacheTTL)

	return response, nil
}

func (uc *transactionUsecase) ProcessPurchase(ctx context.Context, req *transaction.ProcessPurchaseRequest) (*transaction.TransactionResponse, error) {
	if req.GetBuyerId() == "" || req.GetSkinId() == "" {
		return nil, errors.New("buyer_id and skin_id are required")
	}

	buyerID, err := primitive.ObjectIDFromHex(req.GetBuyerId())
	if err != nil {
		return nil, fmt.Errorf("invalid buyer_id: %v", err)
	}

	skinID, err := primitive.ObjectIDFromHex(req.GetSkinId())
	if err != nil {
		return nil, fmt.Errorf("invalid skin_id: %v", err)
	}

	newTransaction := &models.Transaction{
		BuyerID:     buyerID,
		SellerID:    primitive.NewObjectID(),
		SkinID:      skinID,
		Amount:      100.0,
		Status:      models.StatusPending,
		Type:        models.TypeBuy,
		Description: "Purchase transaction",
	}

	createdTransaction, err := uc.transactionRepo.CreateTransaction(ctx, newTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase transaction: %v", err)
	}

	// Invalidate related caches
	uc.invalidateTransactionCaches(createdTransaction.ID.Hex(), &buyerID, &skinID)

	return &transaction.TransactionResponse{
		Transaction: createdTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) CancelTransaction(ctx context.Context, req *transaction.CancelTransactionRequest) (*transaction.TransactionResponse, error) {
	if req.GetId() == "" {
		return nil, errors.New("transaction id is required")
	}

	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid transaction id: %v", err)
	}

	update := bson.M{
		"status": models.StatusCancelled,
	}
	if req.GetReason() != "" {
		update["description"] = fmt.Sprintf("Cancelled: %s", req.GetReason())
	}

	updatedTransaction, err := uc.transactionRepo.UpdateTransaction(ctx, objID, update)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel transaction: %v", err)
	}

	// Invalidate caches after cancellation
	uc.invalidateTransactionCaches(req.GetId(), &updatedTransaction.BuyerID, &updatedTransaction.SkinID)

	return &transaction.TransactionResponse{
		Transaction: updatedTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) GetTransactionStats(ctx context.Context, req *transaction.GetTransactionStatsRequest) (*transaction.TransactionStatsResponse, error) {
	// Generate cache key with parameters
	cacheKey := uc.getListCacheKey(statsCachePrefix, req.GetUserId(), req.GetStartDate(), req.GetEndDate())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if response, ok := cached.(*transaction.TransactionStatsResponse); ok {
			return response, nil
		}
	}

	var userID *primitive.ObjectID
	if req.GetUserId() != "" {
		objID, err := primitive.ObjectIDFromHex(req.GetUserId())
		if err != nil {
			return nil, fmt.Errorf("invalid user_id: %v", err)
		}
		userID = &objID
	}

	stats, err := uc.transactionRepo.GetTransactionStats(ctx, userID, req.GetStartDate(), req.GetEndDate())
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction stats: %v", err)
	}

	response := &transaction.TransactionStatsResponse{
		TotalTransactions:        stats.TotalTransactions,
		TotalAmount:              stats.TotalAmount,
		SuccessfulTransactions:   stats.SuccessfulTransactions,
		FailedTransactions:       stats.FailedTransactions,
		AverageTransactionAmount: stats.AverageAmount,
	}

	// Cache the result with longer TTL since stats are expensive to compute
	uc.cache.Set(cacheKey, response, statsCacheTTL)

	return response, nil
}

func (uc *transactionUsecase) GetAllTransactions(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	// Generate cache key with parameters
	cacheKey := uc.getListCacheKey(allTransactionsCachePrefix, req.GetLimit(), req.GetOffset())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if response, ok := cached.(*transaction.TransactionListResponse); ok {
			return response, nil
		}
	}

	transactions, totalCount, err := uc.transactionRepo.GetAllTransactions(ctx, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	response := &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}

	// Cache the result
	uc.cache.Set(cacheKey, response, listCacheTTL)

	return response, nil
}
