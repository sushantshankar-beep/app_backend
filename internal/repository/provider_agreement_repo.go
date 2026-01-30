// repository/agreement_repository.go
package repository

import (
	"context"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgreementRepo struct {
	collection *mongo.Collection
}

func NewAgreementRepo(db *mongo.Database) *AgreementRepo {
	return &AgreementRepo{
		collection: db.Collection("providerAgreement"),
	}
}

// FindDefault - Get the default/only agreement template
func (r *AgreementRepo) FindDefault(ctx context.Context) (*domain.Agreement, error) {
	var agreement domain.Agreement
	
	// Just get the first (and only) agreement
	err := r.collection.FindOne(ctx, bson.M{}).Decode(&agreement)
	if err != nil {
		return nil, err
	}
	
	return &agreement, nil
}