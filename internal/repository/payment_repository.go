package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentRepository struct {
	txnCol     *mongo.Collection
	webhookCol *mongo.Collection
}

func NewPaymentRepository(db *mongo.Database) *PaymentRepository {
	return &PaymentRepository{
		txnCol:     db.Collection("payment_transactions"),
		webhookCol: db.Collection("payment_webhooks"),
	}
}

func (r *PaymentRepository) CreateTransaction(ctx context.Context, txn *domain.PaymentTransaction) error {
	txn.CreatedAt = time.Now()
	txn.UpdatedAt = time.Now()
	_, err := r.txnCol.InsertOne(ctx, txn)
	return err
}

func (r *PaymentRepository) GetByTxnID(ctx context.Context, txnid string) (*domain.PaymentTransaction, error) {
	var txn domain.PaymentTransaction
	err := r.txnCol.FindOne(ctx, bson.M{"txnid": txnid}).Decode(&txn)
	return &txn, err
}

func (r *PaymentRepository) UpdateTxn(ctx context.Context, txnid string, update bson.M) error {
	update["updatedAt"] = time.Now()
	_, err := r.txnCol.UpdateOne(ctx, bson.M{"txnid": txnid}, bson.M{"$set": update})
	return err
}

func (r *PaymentRepository) SaveWebhook(ctx context.Context, txnid string, payload map[string]interface{}) {
	r.webhookCol.InsertOne(ctx, bson.M{
		"txnid":     txnid,
		"payload":   payload,
		"createdAt": time.Now(),
	})
}
