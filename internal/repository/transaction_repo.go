package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionRepo struct {
	col *mongo.Collection
}

func NewTransactionRepo(db *mongo.Database) *TransactionRepo {
	return &TransactionRepo{
		col: db.Collection("transactions"),
	}
}

func (r *TransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.col.InsertOne(ctx, tx)
	return err
}

func (r *TransactionRepo) UpdateByTxnID(ctx context.Context, txnid string, update map[string]any) error {
	update["updatedAt"] = time.Now()
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"txnid": txnid},
		bson.M{"$set": update},
	)
	return err
}

func (r *TransactionRepo) UpdateByMihPayID(ctx context.Context, mihpayid string, update map[string]any) error {
	update["updatedAt"] = time.Now()
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"mihpayid": mihpayid},
		bson.M{"$set": update},
	)
	return err
}
