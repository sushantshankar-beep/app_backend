package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DiscountRepo struct {
	col *mongo.Collection
}

func NewDiscountRepo(db *mongo.Database) *DiscountRepo {
	return &DiscountRepo{col: db.Collection("discounts")}
}

func (r *DiscountRepo) GetActiveForService(ctx context.Context, applicableOn, zone, paymentMethod string) ([]*domain.Discount, error) {
	now := time.Now()

	filter := bson.M{
		"status":  domain.DiscountStatusActive,
		"startAt": bson.M{"$lte": now},
		"$or": []bson.M{
			{"endAt": nil},
			{"endAt": bson.M{"$gt": now}},
		},
	}

	if applicableOn != "" {
		filter["applicableOn"] = bson.M{
			"$in": []domain.DiscountApplicableOn{
				domain.DiscountApplicableOn(applicableOn),
			},
		}
	}

	if zone != "" {
		filter["zones"] = bson.M{"$in": []string{"all", zone}}
	}

	if paymentMethod != "" {
		filter["paymentMethods"] = bson.M{"$in": []string{"all", paymentMethod}}
	}

	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var discounts []*domain.Discount
	if err := cursor.All(ctx, &discounts); err != nil {
		return nil, err
	}
	return discounts, nil
}