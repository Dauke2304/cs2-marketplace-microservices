package usecase

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/internal/repository"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
	"errors"
)

type InventoryUsecase struct {
	repo repository.InventoryRepository
}

func NewInventoryUsecase(repo repository.InventoryRepository) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

func (uc *InventoryUsecase) CreateSkin(ctx context.Context, req *inventory.CreateSkinRequest) (*inventory.SkinResponse, error) {
	if req.GetSkin().GetPrice() <= 0 {
		return nil, errors.New("price must be positive")
	}
	skin, err := uc.repo.CreateSkin(ctx, req.GetSkin())
	return &inventory.SkinResponse{Skin: skin}, err
}

func (uc *InventoryUsecase) GetSkin(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.SkinResponse, error) {
	skin, err := uc.repo.GetSkin(ctx, req.GetId())
	return &inventory.SkinResponse{Skin: skin}, err
}

func (uc *InventoryUsecase) ListSkins(ctx context.Context, req *inventory.ListSkinsRequest) (*inventory.ListSkinsResponse, error) {
	skins, err := uc.repo.ListSkins(ctx, req.GetOwnerId(), req.GetIsListed())
	return &inventory.ListSkinsResponse{Skins: skins}, err
}

func (uc *InventoryUsecase) UpdateSkin(ctx context.Context, req *inventory.UpdateSkinRequest) (*inventory.SkinResponse, error) {
	if req.GetSkin().GetPrice() <= 0 {
		return nil, errors.New("price must be positive")
	}
	skin, err := uc.repo.UpdateSkin(ctx, req.GetSkin())
	return &inventory.SkinResponse{Skin: skin}, err
}

func (uc *InventoryUsecase) DeleteSkin(ctx context.Context, req *inventory.DeleteSkinRequest) (*inventory.DeleteResponse, error) {
	err := uc.repo.DeleteSkin(ctx, req.GetId())
	return &inventory.DeleteResponse{Success: err == nil}, err
}

func (uc *InventoryUsecase) ToggleListing(ctx context.Context, req *inventory.ToggleListingRequest) (*inventory.SkinResponse, error) {
	err := uc.repo.ToggleListing(ctx, req.GetId(), req.GetIsListed())
	if err != nil {
		return nil, err
	}
	skin, err := uc.repo.GetSkin(ctx, req.GetId())
	return &inventory.SkinResponse{Skin: skin}, err
}

func (uc *InventoryUsecase) TransferOwnership(ctx context.Context, req *inventory.TransferOwnershipRequest) (*inventory.SkinResponse, error) {
	err := uc.repo.TransferOwnership(ctx, req.GetSkinId(), req.GetNewOwnerId())
	if err != nil {
		return nil, err
	}
	skin, err := uc.repo.GetSkin(ctx, req.GetSkinId())
	return &inventory.SkinResponse{Skin: skin}, err
}

func (uc *InventoryUsecase) GetSkinsByOwner(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	skins, err := uc.repo.ListSkins(ctx, req.GetId(), false)
	return &inventory.ListSkinsResponse{Skins: skins}, err
}

func (uc *InventoryUsecase) GetListedSkins(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	skins, err := uc.repo.ListSkins(ctx, "", true)
	return &inventory.ListSkinsResponse{Skins: skins}, err
}
