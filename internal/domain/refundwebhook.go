package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RefundWebhook struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	MihPayID string             `bson:"mihpayid"`
	TxnID    string             `bson:"txnid"`
	Status   string             `bson:"status"`
	Payload  any                `bson:"payload"`
	ReceivedAt time.Time        `bson:"receivedAt"`
}
