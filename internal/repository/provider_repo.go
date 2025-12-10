package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProviderRepo struct {
	col *mongo.Collection
}

func NewProviderRepo(db *mongo.Database) *ProviderRepo {
	return &ProviderRepo{col: db.Collection("providers")}
}

func (r *ProviderRepo) FindByPhone(ctx context.Context, phone string) (*domain.Provider, error) {
	var p domain.Provider
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *ProviderRepo) FindByID(ctx context.Context, id domain.ProviderID) (*domain.Provider, error) {
	objID, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var provider domain.Provider
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&provider)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &provider, nil
}

func (r *ProviderRepo) Create(ctx context.Context, p *domain.Provider) error {
	_, err := r.col.InsertOne(ctx, p)
	return err
}


func (r *ProviderRepo) Update(ctx context.Context, p *domain.Provider) error {
	_, err := r.col.UpdateByID(ctx, p.ID, bson.M{"$set": p})
	return err
}