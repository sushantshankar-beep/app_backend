package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AMCRepo struct {
	col *mongo.Collection
}

func NewAMCRepo(db *mongo.Database) *AMCRepo {
	return &AMCRepo{col: db.Collection("amcs")}
}

func (r *AMCRepo) FindActiveByVehicle(
	ctx context.Context,
	vehicleNumber string,
) (*domain.AMC, error) {

	var amc domain.AMC
	err := r.col.FindOne(ctx, bson.M{
		"vehicleNumber": vehicleNumber,
		"status":        "active",
	}).Decode(&amc)

	if err != nil {
		return nil, err
	}
	return &amc, nil
}
