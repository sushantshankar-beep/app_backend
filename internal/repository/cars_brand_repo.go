package repository

import (
	"context"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CarBrandModelRepo struct {
	col *mongo.Collection
}

func NewCarBrandModelRepo(db *mongo.Database) *CarBrandModelRepo {
	return &CarBrandModelRepo{
		col: db.Collection("CarBrandModel"),
	}
}

func (r *CarBrandModelRepo) GetAll(ctx context.Context, filter bson.M) ([]domain.Brand, error) {
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
