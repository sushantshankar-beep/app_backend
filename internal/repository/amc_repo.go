package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AMCRepo struct {
	col *mongo.Collection
}

func NewAMCRepo(db *mongo.Database) *AMCRepo {
	return &AMCRepo{col: db.Collection("amcs")}
}

func (r *AMCRepo) FindActiveByVehicle(ctx context.Context,vehicleNumber string) (*domain.AMC, error) {
	var amc domain.AMC
	err := r.col.FindOne(ctx, bson.M{"vehicleNumber": vehicleNumber,"isActive":true}).Decode(&amc)
	if err != nil {
		return nil, err
	}
	return &amc, nil
}

func (r *AMCRepo) FindActiveByUser(ctx context.Context,userID primitive.ObjectID) (*domain.AMC, error) {
	var amc domain.AMC
	err := r.col.FindOne(ctx, bson.M{
		"userId":   userID,
		"isActive": "active",
	}).Decode(&amc)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &amc, err
}
