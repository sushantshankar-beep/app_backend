package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type KYCRepo struct {
	col *mongo.Collection
}

func NewKYCRepo(db *mongo.Database) *KYCRepo {
	return &KYCRepo{
		col: db.Collection("provider_kyc"),
	}
}

func (r *KYCRepo) Upsert(ctx context.Context, kyc *domain.ProviderKYC) error {
	filter := bson.M{"providerId": kyc.ProviderID}

	update := bson.M{
		"$set": bson.M{
			"documentType":        kyc.DocumentType,
			"documentUrl":         kyc.DocumentURL,
			"electricityBillUrl":  kyc.ElectricityBillURL,
			"cancelledChequeUrl":  kyc.CancelledChequeURL,
			"accountHolderName":   kyc.AccountHolderName,
			"accountNumber":       kyc.AccountNumber,
			"branchName":          kyc.BranchName,
			"ifsc":                kyc.IFSC,
			"upiId":               kyc.UPIID,
			"gstNumber":           kyc.GSTNumber,
			"status":              kyc.Status,
			"updatedAt":           time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *KYCRepo) FindByProvider(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {
	var kyc domain.ProviderKYC
	err := r.col.FindOne(ctx, bson.M{"providerId": providerID}).Decode(&kyc)
	return &kyc, err
}
