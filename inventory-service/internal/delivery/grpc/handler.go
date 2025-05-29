package grpc

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/internal/usecase"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
)

type Handler struct {
	inventory.UnimplementedInventoryServiceServer
	uc usecase.InventoryUsecase
}

func NewHandler(uc usecase.InventoryUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateSkin(ctx context.Context, req *inventory.CreateSkinRequest) (*inventory.SkinResponse, error) {
	return h.uc.CreateSkin(ctx, req)
}

func (h *Handler) GetSkin(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.SkinResponse, error) {
	return h.uc.GetSkin(ctx, req)
}

func (h *Handler) ListSkins(ctx context.Context, req *inventory.ListSkinsRequest) (*inventory.ListSkinsResponse, error) {
	return h.uc.ListSkins(ctx, req)
}

func (h *Handler) UpdateSkin(ctx context.Context, req *inventory.UpdateSkinRequest) (*inventory.SkinResponse, error) {
	return h.uc.UpdateSkin(ctx, req)
}

func (h *Handler) DeleteSkin(ctx context.Context, req *inventory.DeleteSkinRequest) (*inventory.DeleteResponse, error) {
	return h.uc.DeleteSkin(ctx, req)
}

func (h *Handler) ToggleListing(ctx context.Context, req *inventory.ToggleListingRequest) (*inventory.SkinResponse, error) {
	return h.uc.ToggleListing(ctx, req)
}

func (h *Handler) TransferOwnership(ctx context.Context, req *inventory.TransferOwnershipRequest) (*inventory.SkinResponse, error) {
	return h.uc.TransferOwnership(ctx, req)
}

func (h *Handler) GetSkinsByOwner(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	return h.uc.GetSkinsByOwner(ctx, req)
}

func (h *Handler) GetListedSkins(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	return h.uc.GetListedSkins(ctx, req)
}
