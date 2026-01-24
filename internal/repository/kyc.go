package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type KYCRepo struct {
	collection *mongo.Collection
}

func NewKYCRepo(db *mongo.Database) *KYCRepo {
	return &KYCRepo{
		collection: db.Collection("provider_kyc"),
	}
}

func (r *KYCRepo) Upsert(ctx context.Context, kyc *domain.ProviderKYC) error {
	existing, _ := r.FindByProvider(ctx, kyc.ProviderID.Hex())
	
	now := time.Now()
	kyc.UpdatedAt = now
	
	if existing != nil {
		kyc.ID = existing.ID
		kyc.CreatedAt = existing.CreatedAt
		
		filter := bson.M{"_id": existing.ID}
		update := bson.M{"$set": kyc}
		
		_, err := r.collection.UpdateOne(ctx, filter, update)
		return err
	}
	
	kyc.CreatedAt = now
	result, err := r.collection.InsertOne(ctx, kyc)
	if err != nil {
		return err
	}
	
	kyc.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *KYCRepo) FindByProvider(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {
	objID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}
	
	var kyc domain.ProviderKYC
	err = r.collection.FindOne(ctx, bson.M{"providerId": objID}).Decode(&kyc)
	if err != nil {
		return nil, err
	}
	
	return &kyc, nil
}

func (r *KYCRepo) FindByProviderID(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {

    objID, err := primitive.ObjectIDFromHex(providerID)
    if err != nil {
        return nil, err
    }

    var kyc domain.ProviderKYC
    err = r.collection.FindOne(ctx, bson.M{"providerId": objID}).Decode(&kyc)
    if err == mongo.ErrNoDocuments {
        return nil, nil
    }

    return &kyc, err
}
