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
	skip, limit int64,
) ([]domain.SettlementRecord,int64, error) {

	filter := bson.M{
		"providerId": providerID,
		"settlementStatus": "settled",
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSort(bson.M{"createdAt": -1}).SetSkip(skip).
	SetLimit(limit)

	var records []domain.SettlementRecord

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0,err
	}

	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &records); err != nil {
		return nil,0, err
	}

	if records == nil {
		return []domain.SettlementRecord{}, total, nil
	}

	return records, total, nil
}


func (r *SettlementHistoryRepository) GetTotalSettlementByProvider(
	ctx context.Context,
	providerID primitive.ObjectID,
) (float64, error) {

	filter := bson.M{
		"providerId":       providerID,
		"settlementStatus": "settled",
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	total := 0.0
	for cursor.Next(ctx) {
		var record domain.SettlementRecord
		if err := cursor.Decode(&record); err != nil {
			return 0, err
		}
		total += record.NetAmount
	}

	return total, nil
}
