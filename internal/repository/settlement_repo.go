package repository

import (

	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)


type SettlementHistoryRepository struct {
	collection *mongo.Collection
}

func NewSettlementHistoryRepository(db *mongo.Database) *SettlementHistoryRepository {
	return &SettlementHistoryRepository{collection: db.Collection("settlementHistory")}
}

func (r *SettlementHistoryRepository) GetProviderSettledRecords(
	ctx context.Context,
	providerID primitive.ObjectID,
) ([]domain.SettlementRecord, error) {

	filter := bson.M{
		"providerId": providerID,
		"settlementStatus": "settled",
	}

	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	var records []domain.SettlementRecord

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}
