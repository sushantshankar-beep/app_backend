package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocationRepo struct {
	col *mongo.Collection
}

func NewLocationRepo(db *mongo.Database) *LocationRepo {
	return &LocationRepo{col: db.Collection("locations")}
}


func (r *LocationRepo) Save(ctx context.Context, loc *domain.Location) error {
	filter := bson.M{"userId": loc.UserID}
	update := bson.M{"$set": loc}

	_, err := r.col.UpdateOne(
		ctx,
		filter,
		update,
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *LocationRepo) GetByUser(ctx context.Context, userID string) (*domain.Location, error) {
	var loc domain.Location
	err := r.col.FindOne(ctx, bson.M{"userId": userID}).Decode(&loc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &loc, err
}
