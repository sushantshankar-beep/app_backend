package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserVehicleRepo struct {
	col *mongo.Collection
}

func NewUserVehicleRepo(db *mongo.Database) *UserVehicleRepo {
	return &UserVehicleRepo{
		col: db.Collection("user_vehicles"),
	}
}

/*
UPSERT by userId
*/
func (r *UserVehicleRepo) Upsert(ctx context.Context, uv *domain.UserVehicle) error {
	filter := bson.M{"userId": uv.UserID}

	update := bson.M{
		"$set": bson.M{
			"vehicleId":     uv.VehicleID,
			"vehicleNumber": uv.VehicleNumber,
			"updatedAt":     time.Now(),
		},
		"$setOnInsert": bson.M{
			"userId":    uv.UserID,
			"createdAt": time.Now(),
		},
	}

	_, err := r.col.UpdateOne(
		ctx,
		filter,
		update,
		options.Update().SetUpsert(true),
	)

	return err
}


/*
GET by userId
*/
func (r *UserVehicleRepo) FindByUser(
	ctx context.Context,
	userID primitive.ObjectID,
) (*domain.UserVehicle, error) {

	var v domain.UserVehicle
	err := r.col.FindOne(ctx, bson.M{
		"userId": userID,
	}).Decode(&v)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	return &v, err
}
