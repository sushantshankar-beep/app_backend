package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type VehicleBrandRepo struct {
	col *mongo.Collection
}

func NewVehicleBrandRepo(db *mongo.Database) *VehicleBrandRepo {
	col := db.Collection("vehicleBrands")

	_, _ = col.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys: bson.M{"vehicleType": 1},
		},
	)

	return &VehicleBrandRepo{col: col}
}

func (r *VehicleBrandRepo) FindByVehicle(
	ctx context.Context,
	vehicle string,
) ([]domain.VehicleBrand, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := r.col.Find(ctx, bson.M{
		"vehicleType": vehicle,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var brands []domain.VehicleBrand
	if err := cur.All(ctx, &brands); err != nil {
		return nil, err
	}

	return brands, nil
}
