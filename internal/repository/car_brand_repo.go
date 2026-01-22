package repository

import (
	"context"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BikeBrandModelRepo struct {
	col *mongo.Collection
}

func NewBikeBrandModelRepo(db *mongo.Database) *BikeBrandModelRepo {
	return &BikeBrandModelRepo{
		col: db.Collection("BikeBrandModel"),
	}
}

func (r *BikeBrandModelRepo) GetAll(ctx context.Context, filter bson.M) ([]domain.Brand, error) {
	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var brands []domain.Brand
	if err := cursor.All(ctx, &brands); err != nil {
		return nil, err
	}

	return brands, nil
}
