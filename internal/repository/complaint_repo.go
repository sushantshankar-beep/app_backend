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
