package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AgreementRepo struct {
	col *mongo.Collection
}

func NewAgreementRepo(db *mongo.Database) *AgreementRepo {
	return &AgreementRepo{col: db.Collection("providerAgreement")}
}

func (r *AgreementRepo) FindByID(ctx context.Context, id string) (*domain.Agreement, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var agreement domain.Agreement
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&agreement)
	if err != nil {
		return nil, err
	}
	return &agreement, nil
}
