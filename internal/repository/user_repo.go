package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepo struct {
	col *mongo.Collection
}

func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{col: db.Collection("users")}
}
func (r *UserRepo) AddComplaint(ctx context.Context, userID primitive.ObjectID, complaintID primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, userID, bson.M{
		"$push": bson.M{"complaintsSubmitted": complaintID},
	})
	return err
}

func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var u domain.User
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.col.InsertOne(ctx, u)
	return err
}
