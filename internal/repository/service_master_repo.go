package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServiceMasterRepo struct {
	col *mongo.Collection
}

func NewServiceMasterRepo(db *mongo.Database) *ServiceMasterRepo {
	col := db.Collection("serviceMaster")

	_, _ = col.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys: bson.M{"vehicleType": 1},
		},
	)

	return &ServiceMasterRepo{col: col}
}

func (r *ServiceMasterRepo) FindByVehicle(
	ctx context.Context,
	vehicle string,
) ([]domain.ServiceMaster, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := r.col.Find(ctx, bson.M{
		"vehicleType": vehicle,
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var services []domain.ServiceMaster
	if err := cur.All(ctx, &services); err != nil {
		return nil, err
	}

	return services, nil
}
