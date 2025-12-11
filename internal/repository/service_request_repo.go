package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServiceRequestRepo struct {
	col *mongo.Collection
}

func NewServiceRequestRepo(db *mongo.Database) *ServiceRequestRepo {
	return &ServiceRequestRepo{col: db.Collection("servicerequests")}
}

func (r *ServiceRequestRepo) Create(ctx context.Context, s *domain.ServiceRequest) error {
	res, err := r.col.InsertOne(ctx, s)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		s.ID = oid
	}
	return nil
}

func (r *ServiceRequestRepo) Update(ctx context.Context, s *domain.ServiceRequest) error {
	update := bson.M{
		"$set": bson.M{
			"expiresAt": s.ExpiresAt,
			"updatedAt": s.UpdatedAt,
		},
	}

	_, err := r.col.UpdateByID(ctx, s.ID, update)
	return err
}

func (r *ServiceRequestRepo) FindByID(ctx context.Context, id string) (*domain.ServiceRequest, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var serviceRequest domain.ServiceRequest
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&serviceRequest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &serviceRequest, nil
}

func (r *ServiceRequestRepo) FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.ServiceRequest, error) {
	var serviceRequest domain.ServiceRequest
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&serviceRequest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &serviceRequest, nil
}

type SavedVehicleRepo struct {
	col *mongo.Collection
}

func NewSavedVehicleRepo(db *mongo.Database) *SavedVehicleRepo {
	return &SavedVehicleRepo{col: db.Collection("savedvehicles")}
}

func (r *SavedVehicleRepo) FindByUserAndVehicleNumber(ctx context.Context, userID string, vehicleNumber string) (*domain.SavedVehicleData, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var vehicle domain.SavedVehicleData
	err = r.col.FindOne(ctx, bson.M{
		"userId":        objID,
		"vehicleNumber": vehicleNumber,
	}).Decode(&vehicle)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &vehicle, nil
}

func (r *SavedVehicleRepo) Create(ctx context.Context, v *domain.SavedVehicleData) error {
	res, err := r.col.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		v.ID = oid
	}
	return nil
}
