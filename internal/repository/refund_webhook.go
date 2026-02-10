package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RefundWebhookRepo struct {
	col *mongo.Collection
}

// Constructor
func NewRefundWebhookRepo(db *mongo.Database) *RefundWebhookRepo {
	return &RefundWebhookRepo{
		col: db.Collection("refund_webhooks"),
	}
}

// ---------------------------------------
// Save raw refund webhook payload
// ---------------------------------------

func (r *RefundWebhookRepo) SaveWebhook(
	ctx context.Context,
	mihpayid string,
	payload map[string]any,
) error {

	doc := bson.M{
		"mihpayid": mihpayid,
		"payload":  payload,
		"createdAt": time.Now(),
	}

	_, err := r.col.InsertOne(ctx, doc)
	return err
}
