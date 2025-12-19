package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServiceCatalogRepo struct {
	col *mongo.Collection
}

func NewServiceCatalogRepo(db *mongo.Database) *ServiceCatalogRepo {
	return &ServiceCatalogRepo{
		col: db.Collection("service_catalog"),
	}
}

func (r *ServiceCatalogRepo) FindByName(
	ctx context.Context,
	name string,
) (*domain.ServiceCatalog, error) {

	var svc domain.ServiceCatalog
	if err := r.col.FindOne(ctx, bson.M{"name": name}).Decode(&svc); err != nil {
		return nil, err
	}
	return &svc, nil
}
