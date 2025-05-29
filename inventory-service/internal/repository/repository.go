package repository

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
)

type InventoryRepository interface {
	CreateSkin(ctx context.Context, skin *inventory.Skin) (*inventory.Skin, error)
	GetSkin(ctx context.Context, id string) (*inventory.Skin, error)
	ListSkins(ctx context.Context, ownerID string, isListed bool) ([]*inventory.Skin, error)
	UpdateSkin(ctx context.Context, skin *inventory.Skin) (*inventory.Skin, error)
	DeleteSkin(ctx context.Context, id string) error
	ToggleListing(ctx context.Context, id string, isListed bool) error
	TransferOwnership(ctx context.Context, skinID, newOwnerID string) error
}
