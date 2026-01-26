package repository

import (
	"app_backend/internal/domain"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ComplaintRepo struct {
	col *mongo.Collection
}

func NewComplaintRepo(db *mongo.Database) *ComplaintRepo {
	return &ComplaintRepo{col: db.Collection("complaints")}
}

func (r *ComplaintRepo) Create(ctx context.Context, c *domain.Complaint) error {
	_, err := r.col.InsertOne(ctx, c)
	return err
}

func (r *ComplaintRepo) Update(ctx context.Context, c *domain.Complaint) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": c.ID}, c)
	return err
}

func (r *ComplaintRepo) FindByAcceptedServiceId(ctx context.Context, acceptedServiceId primitive.ObjectID) (*domain.Complaint, error) {
	var complaint domain.Complaint
	err := r.col.FindOne(ctx, bson.M{"acceptedService": acceptedServiceId}).Decode(&complaint)
	if err != nil {
		return nil, err
	}
	return &complaint, nil
}

func (r *ComplaintRepo) FindByUser(ctx context.Context, userID primitive.ObjectID) ([]domain.Complaint, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}

	var list []domain.Complaint
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ComplaintRepo) FindByProvider(ctx context.Context, providerID primitive.ObjectID) ([]domain.Complaint, error) {
	cur, err := r.col.Find(ctx, bson.M{"providerId": providerID})
	if err != nil {
		return nil, err
	}

	var list []domain.Complaint
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

