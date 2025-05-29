package usecase

import (
	"context"
	"cs2-marketplace-microservices/transaction-service/internal/models"
	"cs2-marketplace-microservices/transaction-service/internal/repository"
	"cs2-marketplace-microservices/transaction-service/proto/transaction"
	"errors"
	"fmt"

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
}

func NewTransactionUsecase(transactionRepo repository.TransactionRepository) TransactionUsecase {
	return &transactionUsecase{
		transactionRepo: transactionRepo,
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

	return &transaction.TransactionResponse{
		Transaction: createdTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) GetTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.TransactionResponse, error) {
	if req.GetId() == "" {
		return nil, errors.New("transaction id is required")
	}

	objID, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid transaction id: %v", err)
	}

	trans, err := uc.transactionRepo.GetTransactionByID(ctx, objID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

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

	err = uc.transactionRepo.DeleteTransaction(ctx, objID)
	if err != nil {
		return &transaction.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("failed to delete transaction: %v", err),
		}, nil
	}

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

	return &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}, nil
}

func (uc *transactionUsecase) GetTransactionsBySkin(ctx context.Context, req *transaction.GetTransactionsBySkinRequest) (*transaction.TransactionListResponse, error) {
	if req.GetSkinId() == "" {
		return nil, errors.New("skin id is required")
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

	return &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(len(transactions)),
	}, nil
}

func (uc *transactionUsecase) GetTransactionsByStatus(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	status := models.StatusFromProto(req.GetStatus())

	transactions, totalCount, err := uc.transactionRepo.GetTransactionsByStatus(ctx, status, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	return &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}, nil
}

func (uc *transactionUsecase) ProcessPurchase(ctx context.Context, req *transaction.ProcessPurchaseRequest) (*transaction.TransactionResponse, error) {
	if req.GetBuyerId() == "" || req.GetSkinId() == "" {
		return nil, errors.New("buyer_id and skin_id are required")
	}

	// For now, we'll create a basic purchase transaction
	// Later this will integrate with inventory and user services
	buyerID, err := primitive.ObjectIDFromHex(req.GetBuyerId())
	if err != nil {
		return nil, fmt.Errorf("invalid buyer_id: %v", err)
	}

	skinID, err := primitive.ObjectIDFromHex(req.GetSkinId())
	if err != nil {
		return nil, fmt.Errorf("invalid skin_id: %v", err)
	}

	// TODO: When user and inventory services are ready:
	// 1. Get skin details from inventory service
	// 2. Check user balance from user service
	// 3. Transfer ownership via inventory service
	// 4. Update user balance via user service

	// For now, create a basic transaction with dummy data
	newTransaction := &models.Transaction{
		BuyerID:     buyerID,
		SellerID:    primitive.NewObjectID(), // Dummy seller ID for now
		SkinID:      skinID,
		Amount:      100.0, // Dummy amount for now
		Status:      models.StatusPending,
		Type:        models.TypeBuy,
		Description: "Purchase transaction",
	}

	createdTransaction, err := uc.transactionRepo.CreateTransaction(ctx, newTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase transaction: %v", err)
	}

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

	return &transaction.TransactionResponse{
		Transaction: updatedTransaction.ToProto(),
	}, nil
}

func (uc *transactionUsecase) GetTransactionStats(ctx context.Context, req *transaction.GetTransactionStatsRequest) (*transaction.TransactionStatsResponse, error) {
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

	return &transaction.TransactionStatsResponse{
		TotalTransactions:        stats.TotalTransactions,
		TotalAmount:              stats.TotalAmount,
		SuccessfulTransactions:   stats.SuccessfulTransactions,
		FailedTransactions:       stats.FailedTransactions,
		AverageTransactionAmount: stats.AverageAmount,
	}, nil
}

func (uc *transactionUsecase) GetAllTransactions(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	transactions, totalCount, err := uc.transactionRepo.GetAllTransactions(ctx, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions: %v", err)
	}

	protoTransactions := make([]*transaction.Transaction, len(transactions))
	for i, t := range transactions {
		protoTransactions[i] = t.ToProto()
	}

	return &transaction.TransactionListResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(totalCount),
	}, nil
}
