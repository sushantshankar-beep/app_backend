package repository

import (
	"context"
	"time"
	"go.mongodb.org/mongo-driver/mongo/options"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"fmt"
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
func (r *PaymentRepository) GetLatestByServiceID(
	ctx context.Context,
	serviceID string,
) (*domain.PaymentTransaction, error) {

	var txn domain.PaymentTransaction

	err := r.txnCol.FindOne(
		ctx,
		bson.M{"serviceId": serviceID},
		options.FindOne().SetSort(bson.M{"createdAt": -1}),
	).Decode(&txn)

	if err != nil {
		return nil, err
	}

	return &txn, nil
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
	fmt.Println(err)
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

func (r *PaymentRepository) GetTransactionByServiceID(ctx context.Context, serviceID string) (*domain.PaymentTransaction, error) {
	var transaction domain.PaymentTransaction
	
	filter := bson.M{"serviceId": serviceID}
	
	opts := options.FindOne().
	SetSort(bson.D{{Key: "createdAt", Value: -1}})

	err := r.txnCol.FindOne(ctx, filter,opts).Decode(&transaction)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("transaction not found for serviceID: %s", serviceID)
		}
		return nil, err
	}
	
	return &transaction, nil
}
func (r *PaymentRepository) FindByServiceID(
	ctx context.Context,
	serviceID string,
) (*domain.PaymentTransaction, error) {

	var txn domain.PaymentTransaction

	err := r.txnCol.FindOne(ctx, bson.M{
		"serviceId": serviceID,
		"status":    "paid",
	}).Decode(&txn)

	return &txn, err
}

func (r *PaymentRepository) GetLatestPaidTransactionByServiceID(
	ctx context.Context,
	serviceID string,
) (*domain.PaymentTransaction, error) {

	filter := bson.M{
		"serviceId": serviceID,
		"status":    "paid",
	}

	opts := options.FindOne().
		SetSort(bson.D{{Key: "createdAt", Value: -1}})

	var txn domain.PaymentTransaction
	err := r.txnCol.FindOne(ctx, filter, opts).Decode(&txn)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no paid transaction found for service")
		}
		return nil, err
	}

	return &txn, nil
}
func (r *PaymentRepository) MarkInvoiceGenerated(
	ctx context.Context,
	txnid string,
) (bool, error) {

	res, err := r.txnCol.UpdateOne(
		ctx,
		bson.M{
			"txnid":           txnid,
			"invoiceGenerated": false,
			"status":          "paid", // safety
		},
		bson.M{
			"$set": bson.M{
				"invoiceGenerated":   true,
				"invoiceGeneratedAt": time.Now(),
				"updatedAt":          time.Now(),
			},
		},
	)

	if err != nil {
		return false, err
	}

	return res.ModifiedCount == 1, nil
}
