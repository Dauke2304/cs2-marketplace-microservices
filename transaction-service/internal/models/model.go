package models

import (
	"cs2-marketplace-microservices/transaction-service/proto/transaction"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionStatus string
type TransactionType string

const (
	StatusPending   TransactionStatus = "PENDING"
	StatusCompleted TransactionStatus = "COMPLETED"
	StatusFailed    TransactionStatus = "FAILED"
	StatusCancelled TransactionStatus = "CANCELLED"
)

const (
	TypeBuy  TransactionType = "BUY"
	TypeSell TransactionType = "SELL"
)

type Transaction struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	BuyerID     primitive.ObjectID `bson:"buyer_id"`
	SellerID    primitive.ObjectID `bson:"seller_id"`
	SkinID      primitive.ObjectID `bson:"skin_id"`
	Amount      float64            `bson:"amount"`
	Date        string             `bson:"date"`
	Status      TransactionStatus  `bson:"status"`
	Type        TransactionType    `bson:"type"`
	Description string             `bson:"description"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

// Converts MongoDB model to Protobuf message
func (t *Transaction) ToProto() *transaction.Transaction {
	return &transaction.Transaction{
		Id:          t.ID.Hex(),
		BuyerId:     t.BuyerID.Hex(),
		SellerId:    t.SellerID.Hex(),
		SkinId:      t.SkinID.Hex(),
		Amount:      t.Amount,
		Date:        t.Date,
		Status:      protoStatusFromString(string(t.Status)),
		Type:        protoTypeFromString(string(t.Type)),
		Description: t.Description,
	}
}

// Converts Protobuf message to MongoDB model
func TransactionFromProto(p *transaction.Transaction) (*Transaction, error) {
	var objID primitive.ObjectID
	var err error

	if p.GetId() != "" {
		objID, err = primitive.ObjectIDFromHex(p.GetId())
		if err != nil {
			return nil, err
		}
	}

	buyerID, err := primitive.ObjectIDFromHex(p.GetBuyerId())
	if err != nil {
		return nil, err
	}

	sellerID, err := primitive.ObjectIDFromHex(p.GetSellerId())
	if err != nil {
		return nil, err
	}

	skinID, err := primitive.ObjectIDFromHex(p.GetSkinId())
	if err != nil {
		return nil, err
	}

	return &Transaction{
		ID:          objID,
		BuyerID:     buyerID,
		SellerID:    sellerID,
		SkinID:      skinID,
		Amount:      p.GetAmount(),
		Date:        p.GetDate(),
		Status:      TransactionStatus(p.GetStatus().String()),
		Type:        TransactionType(p.GetType().String()),
		Description: p.GetDescription(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// Helper functions to convert between proto enums and strings
func protoStatusFromString(status string) transaction.TransactionStatus {
	switch status {
	case "PENDING":
		return transaction.TransactionStatus_PENDING
	case "COMPLETED":
		return transaction.TransactionStatus_COMPLETED
	case "FAILED":
		return transaction.TransactionStatus_FAILED
	case "CANCELLED":
		return transaction.TransactionStatus_CANCELLED
	default:
		return transaction.TransactionStatus_PENDING
	}
}

func protoTypeFromString(txType string) transaction.TransactionType {
	switch txType {
	case "BUY":
		return transaction.TransactionType_BUY
	case "SELL":
		return transaction.TransactionType_SELL
	default:
		return transaction.TransactionType_BUY
	}
}

// Helper functions to convert from proto enums to strings
func StatusFromProto(status transaction.TransactionStatus) TransactionStatus {
	switch status {
	case transaction.TransactionStatus_PENDING:
		return StatusPending
	case transaction.TransactionStatus_COMPLETED:
		return StatusCompleted
	case transaction.TransactionStatus_FAILED:
		return StatusFailed
	case transaction.TransactionStatus_CANCELLED:
		return StatusCancelled
	default:
		return StatusPending
	}
}

func TypeFromProto(txType transaction.TransactionType) TransactionType {
	switch txType {
	case transaction.TransactionType_BUY:
		return TypeBuy
	case transaction.TransactionType_SELL:
		return TypeSell
	default:
		return TypeBuy
	}
}
