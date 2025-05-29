package mongo

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/internal/models"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InventoryRepository struct {
	collection *mongo.Collection
}

func NewInventoryRepository(db *mongo.Database) *InventoryRepository {
	return &InventoryRepository{
		collection: db.Collection("skins"),
	}
}

func (r *InventoryRepository) CreateSkin(ctx context.Context, skin *inventory.Skin) (*inventory.Skin, error) {
	modelSkin, err := models.SkinFromProto(skin)
	if err != nil {
		return nil, err
	}

	modelSkin.CreatedAt = time.Now()
	modelSkin.UpdatedAt = time.Now()

	res, err := r.collection.InsertOne(ctx, modelSkin)
	if err != nil {
		return nil, err
	}

	modelSkin.ID = res.InsertedID.(primitive.ObjectID)
	return modelSkin.ToProto(), nil
}

func (r *InventoryRepository) GetSkin(ctx context.Context, id string) (*inventory.Skin, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid skin ID format")
	}

	var skin models.Skin
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&skin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("skin not found")
		}
		return nil, err
	}

	return skin.ToProto(), nil
}

func (r *InventoryRepository) ListSkins(ctx context.Context, ownerID string, isListed bool) ([]*inventory.Skin, error) {
	filter := bson.M{}
	if ownerID != "" {
		objID, err := primitive.ObjectIDFromHex(ownerID)
		if err != nil {
			return nil, errors.New("invalid owner ID format")
		}
		filter["owner_id"] = objID
	}
	if isListed {
		filter["is_listed"] = true
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var skins []*inventory.Skin
	for cursor.Next(ctx) {
		var skin models.Skin
		if err := cursor.Decode(&skin); err != nil {
			return nil, err
		}
		skins = append(skins, skin.ToProto())
	}

	return skins, nil
}

func (r *InventoryRepository) UpdateSkin(ctx context.Context, skin *inventory.Skin) (*inventory.Skin, error) {
	objID, err := primitive.ObjectIDFromHex(skin.GetId())
	if err != nil {
		return nil, errors.New("invalid skin ID format")
	}

	update := bson.M{
		"$set": bson.M{
			"name":        skin.GetName(),
			"description": skin.GetDescription(),
			"price":       skin.GetPrice(),
			"image":       skin.GetImage(),
			"rarity":      skin.GetRarity(),
			"condition":   skin.GetCondition(),
			"is_listed":   skin.GetIsListed(),
			"updated_at":  time.Now(),
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedSkin models.Skin
	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		update,
		opts,
	).Decode(&updatedSkin)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("skin not found")
		}
		return nil, err
	}

	return updatedSkin.ToProto(), nil
}

func (r *InventoryRepository) DeleteSkin(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid skin ID format")
	}

	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("skin not found")
	}

	return nil
}

func (r *InventoryRepository) ToggleListing(ctx context.Context, id string, isListed bool) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid skin ID format")
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{
			"is_listed":  isListed,
			"updated_at": time.Now(),
		}},
	)
	return err
}

func (r *InventoryRepository) TransferOwnership(ctx context.Context, skinID, newOwnerID string) error {
	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		skinObjID, err := primitive.ObjectIDFromHex(skinID)
		if err != nil {
			return nil, errors.New("invalid skin ID format")
		}

		ownerObjID, err := primitive.ObjectIDFromHex(newOwnerID)
		if err != nil {
			return nil, errors.New("invalid owner ID format")
		}

		_, err = r.collection.UpdateOne(
			sessCtx,
			bson.M{"_id": skinObjID},
			bson.M{"$set": bson.M{
				"owner_id":   ownerObjID,
				"is_listed":  false,
				"updated_at": time.Now(),
			}},
		)
		return nil, err
	})
	return err
}
