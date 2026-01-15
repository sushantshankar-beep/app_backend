package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BidRepo struct {
	col *mongo.Collection
}

func NewBidRepo(db *mongo.Database) *BidRepo {
	return &BidRepo{
		col: db.Collection("bids"),
	}
}

func (r *BidRepo) Insert(ctx context.Context, bid *domain.BidLog) (primitive.ObjectID, error) {
	res, err := r.col.InsertOne(ctx, bid)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}
