package grpc

import (
	"context"
	"cs2-marketplace-microservices/transaction-service/internal/usecase"
	"cs2-marketplace-microservices/transaction-service/proto/transaction"
)

type Handler struct {
	transaction.UnimplementedTransactionServiceServer
	uc usecase.TransactionUsecase
}

func NewHandler(uc usecase.TransactionUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateTransaction(ctx context.Context, req *transaction.CreateTransactionRequest) (*transaction.TransactionResponse, error) {
	return h.uc.CreateTransaction(ctx, req)
}

func (h *Handler) GetTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.TransactionResponse, error) {
	return h.uc.GetTransaction(ctx, req)
}

func (h *Handler) UpdateTransaction(ctx context.Context, req *transaction.UpdateTransactionRequest) (*transaction.TransactionResponse, error) {
	return h.uc.UpdateTransaction(ctx, req)
}

func (h *Handler) DeleteTransaction(ctx context.Context, req *transaction.GetTransactionRequest) (*transaction.DeleteResponse, error) {
	return h.uc.DeleteTransaction(ctx, req)
}

func (h *Handler) ListTransactions(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error) {
	return h.uc.ListTransactions(ctx, req)
}

func (h *Handler) GetTransactionsByUser(ctx context.Context, req *transaction.GetTransactionsByUserRequest) (*transaction.TransactionListResponse, error) {
	return h.uc.GetTransactionsByUser(ctx, req)
}

func (h *Handler) GetTransactionsBySkin(ctx context.Context, req *transaction.GetTransactionsBySkinRequest) (*transaction.TransactionListResponse, error) {
	return h.uc.GetTransactionsBySkin(ctx, req)
}

func (h *Handler) GetTransactionsByStatus(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	return h.uc.GetTransactionsByStatus(ctx, req)
}

func (h *Handler) ProcessPurchase(ctx context.Context, req *transaction.ProcessPurchaseRequest) (*transaction.TransactionResponse, error) {
	return h.uc.ProcessPurchase(ctx, req)
}

func (h *Handler) CancelTransaction(ctx context.Context, req *transaction.CancelTransactionRequest) (*transaction.TransactionResponse, error) {
	return h.uc.CancelTransaction(ctx, req)
}

func (h *Handler) GetTransactionStats(ctx context.Context, req *transaction.GetTransactionStatsRequest) (*transaction.TransactionStatsResponse, error) {
	return h.uc.GetTransactionStats(ctx, req)
}

func (h *Handler) GetAllTransactions(ctx context.Context, req *transaction.GetTransactionsByStatusRequest) (*transaction.TransactionListResponse, error) {
	return h.uc.GetAllTransactions(ctx, req)
}
