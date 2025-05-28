package models

import (
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Skin struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	Price       float64            `bson:"price"`
	Image       string             `bson:"image"`
	Rarity      string             `bson:"rarity"`
	Condition   string             `bson:"condition"`
	OwnerID     primitive.ObjectID `bson:"owner_id,omitempty"`
	IsListed    bool               `bson:"is_listed"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

// Converts MongoDB model to Protobuf message
func (s *Skin) ToProto() *inventory.Skin {
	return &inventory.Skin{
		Id:          s.ID.Hex(),
		Name:        s.Name,
		Description: s.Description,
		Price:       s.Price,
		Image:       s.Image,
		Rarity:      s.Rarity,
		Condition:   s.Condition,
		OwnerId:     s.OwnerID.Hex(),
		IsListed:    s.IsListed,
	}
}

// Converts Protobuf message to MongoDB model
func SkinFromProto(p *inventory.Skin) (*Skin, error) {
	var objID primitive.ObjectID
	var err error

	if p.GetId() != "" {
		objID, err = primitive.ObjectIDFromHex(p.GetId())
		if err != nil {
			return nil, err
		}
	}

	var ownerID primitive.ObjectID
	if p.GetOwnerId() != "" {
		ownerID, err = primitive.ObjectIDFromHex(p.GetOwnerId())
		if err != nil {
			return nil, err
		}
	}

	return &Skin{
		ID:          objID,
		Name:        p.GetName(),
		Description: p.GetDescription(),
		Price:       p.GetPrice(),
		Image:       p.GetImage(),
		Rarity:      p.GetRarity(),
		Condition:   p.GetCondition(),
		OwnerID:     ownerID,
		IsListed:    p.GetIsListed(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}
