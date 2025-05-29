package usecase

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/internal/repository"
	"cs2-marketplace-microservices/inventory-service/pkg/messaging"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/patrickmn/go-cache"
)

type InventoryUsecase struct {
	repo  repository.InventoryRepository
	nats  *messaging.Client
	cache *cache.Cache
}

func NewInventoryUsecase(repo repository.InventoryRepository, nats *messaging.Client) *InventoryUsecase {
	log.Printf("Initializing usecase with NATS client: %v", nats)

	// Initialize cache with 5 minute default expiration and 10 minute cleanup interval
	c := cache.New(5*time.Minute, 10*time.Minute)

	return &InventoryUsecase{
		repo:  repo,
		nats:  nats,
		cache: c,
	}
}

func (uc *InventoryUsecase) CreateSkin(ctx context.Context, req *inventory.CreateSkinRequest) (*inventory.SkinResponse, error) {
	if req.GetSkin().GetPrice() <= 0 {
		return nil, errors.New("price must be positive")
	}

	skin, err := uc.repo.CreateSkin(ctx, req.GetSkin())
	if err != nil {
		return nil, err
	}

	// Cache the newly created skin
	cacheKey := fmt.Sprintf("skin:%s", skin.GetId())
	uc.cache.Set(cacheKey, skin, cache.DefaultExpiration)

	uc.invalidateListCaches(skin.GetOwnerId())

	if uc.nats != nil && uc.nats.Conn != nil {
		log.Printf("Publishing skin ID: %s to NATS", skin.GetId())
		if pubErr := uc.nats.Conn.Publish("skin.created", []byte(skin.GetId())); pubErr != nil {
			log.Printf("NATS publish error: %v", pubErr)
		}
	}

	return &inventory.SkinResponse{Skin: skin}, nil
}

func (uc *InventoryUsecase) GetSkin(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.SkinResponse, error) {
	cacheKey := fmt.Sprintf("skin:%s", req.GetId())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if skin, ok := cached.(*inventory.Skin); ok {
			log.Printf("Cache hit for skin: %s", req.GetId())
			return &inventory.SkinResponse{Skin: skin}, nil
		}
	}

	// Cache miss - get from database
	log.Printf("Cache miss for skin: %s", req.GetId())
	skin, err := uc.repo.GetSkin(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Store in cache
	uc.cache.Set(cacheKey, skin, cache.DefaultExpiration)

	return &inventory.SkinResponse{Skin: skin}, nil
}

func (uc *InventoryUsecase) ListSkins(ctx context.Context, req *inventory.ListSkinsRequest) (*inventory.ListSkinsResponse, error) {
	cacheKey := fmt.Sprintf("list:%s:%t", req.GetOwnerId(), req.GetIsListed())

	// Try to get from cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if skins, ok := cached.([]*inventory.Skin); ok {
			log.Printf("Cache hit for list: %s", cacheKey)
			return &inventory.ListSkinsResponse{Skins: skins}, nil
		}
	}

	// Cache miss - get from database
	log.Printf("Cache miss for list: %s", cacheKey)
	skins, err := uc.repo.ListSkins(ctx, req.GetOwnerId(), req.GetIsListed())
	if err != nil {
		return nil, err
	}

	// Store in cache with shorter expiration for lists (2 minutes)
	uc.cache.Set(cacheKey, skins, 2*time.Minute)

	return &inventory.ListSkinsResponse{Skins: skins}, nil
}

func (uc *InventoryUsecase) UpdateSkin(ctx context.Context, req *inventory.UpdateSkinRequest) (*inventory.SkinResponse, error) {
	if req.GetSkin().GetPrice() <= 0 {
		return nil, errors.New("price must be positive")
	}

	skin, err := uc.repo.UpdateSkin(ctx, req.GetSkin())
	if err != nil {
		return nil, err
	}

	// Update cache with new data
	cacheKey := fmt.Sprintf("skin:%s", skin.GetId())
	uc.cache.Set(cacheKey, skin, cache.DefaultExpiration)

	// Invalidate list caches since skin data changed
	uc.invalidateListCaches(skin.GetOwnerId())

	return &inventory.SkinResponse{Skin: skin}, nil
}

func (uc *InventoryUsecase) DeleteSkin(ctx context.Context, req *inventory.DeleteSkinRequest) (*inventory.DeleteResponse, error) {
	// Get skin before deletion to know owner for cache invalidation
	skin, _ := uc.repo.GetSkin(ctx, req.GetId())

	err := uc.repo.DeleteSkin(ctx, req.GetId())
	if err != nil {
		return &inventory.DeleteResponse{Success: false}, err
	}

	// Remove from cache
	cacheKey := fmt.Sprintf("skin:%s", req.GetId())
	uc.cache.Delete(cacheKey)

	// Invalidate list caches
	if skin != nil {
		uc.invalidateListCaches(skin.GetOwnerId())
	}

	return &inventory.DeleteResponse{Success: true}, nil
}

func (uc *InventoryUsecase) ToggleListing(ctx context.Context, req *inventory.ToggleListingRequest) (*inventory.SkinResponse, error) {
	err := uc.repo.ToggleListing(ctx, req.GetId(), req.GetIsListed())
	if err != nil {
		return nil, err
	}

	// Get updated skin from database and update cache
	skin, err := uc.repo.GetSkin(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Update cache
	cacheKey := fmt.Sprintf("skin:%s", req.GetId())
	uc.cache.Set(cacheKey, skin, cache.DefaultExpiration)

	// Invalidate list caches since listing status changed
	uc.invalidateListCaches(skin.GetOwnerId())

	return &inventory.SkinResponse{Skin: skin}, nil
}

func (uc *InventoryUsecase) TransferOwnership(ctx context.Context, req *inventory.TransferOwnershipRequest) (*inventory.SkinResponse, error) {
	// Get current skin to know old owner for cache invalidation
	oldSkin, _ := uc.repo.GetSkin(ctx, req.GetSkinId())

	err := uc.repo.TransferOwnership(ctx, req.GetSkinId(), req.GetNewOwnerId())
	if err != nil {
		return nil, err
	}

	// Get updated skin
	skin, err := uc.repo.GetSkin(ctx, req.GetSkinId())
	if err != nil {
		return nil, err
	}

	// Update cache
	cacheKey := fmt.Sprintf("skin:%s", req.GetSkinId())
	uc.cache.Set(cacheKey, skin, cache.DefaultExpiration)

	// Invalidate list caches for both old and new owners
	if oldSkin != nil {
		uc.invalidateListCaches(oldSkin.GetOwnerId())
	}
	uc.invalidateListCaches(req.GetNewOwnerId())

	return &inventory.SkinResponse{Skin: skin}, nil
}

func (uc *InventoryUsecase) GetSkinsByOwner(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	cacheKey := fmt.Sprintf("list:%s:false", req.GetId())

	// Try cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if skins, ok := cached.([]*inventory.Skin); ok {
			log.Printf("Cache hit for owner skins: %s", req.GetId())
			return &inventory.ListSkinsResponse{Skins: skins}, nil
		}
	}

	// Cache miss
	skins, err := uc.repo.ListSkins(ctx, req.GetId(), false)
	if err != nil {
		return nil, err
	}

	// Cache with shorter expiration
	uc.cache.Set(cacheKey, skins, 2*time.Minute)

	return &inventory.ListSkinsResponse{Skins: skins}, nil
}

func (uc *InventoryUsecase) GetListedSkins(ctx context.Context, req *inventory.GetSkinRequest) (*inventory.ListSkinsResponse, error) {
	cacheKey := "list::true" // Empty owner ID means all listed skins

	// Try cache first
	if cached, found := uc.cache.Get(cacheKey); found {
		if skins, ok := cached.([]*inventory.Skin); ok {
			log.Printf("Cache hit for listed skins")
			return &inventory.ListSkinsResponse{Skins: skins}, nil
		}
	}

	// Cache miss
	skins, err := uc.repo.ListSkins(ctx, "", true)
	if err != nil {
		return nil, err
	}

	// Cache with shorter expiration for frequently changing data
	uc.cache.Set(cacheKey, skins, 1*time.Minute)

	return &inventory.ListSkinsResponse{Skins: skins}, nil
}

// Helper function to invalidate list caches when data changes
func (uc *InventoryUsecase) invalidateListCaches(ownerID string) {
	// Invalidate all list caches for this owner
	uc.cache.Delete(fmt.Sprintf("list:%s:true", ownerID))
	uc.cache.Delete(fmt.Sprintf("list:%s:false", ownerID))

	// Invalidate global listed skins cache
	uc.cache.Delete("list::true")

	log.Printf("Invalidated list caches for owner: %s", ownerID)
}
