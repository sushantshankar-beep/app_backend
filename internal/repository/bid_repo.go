package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BidRepo struct {
	col *mongo.Collection
}

func NewBidRepo(db *mongo.Database) *BidRepo {
	return &BidRepo{col: db.Collection("bids")}
}

func (r *BidRepo) FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.Bid, error) {
	var bid domain.Bid
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&bid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &bid, nil
}
