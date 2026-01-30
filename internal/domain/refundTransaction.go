package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RefundTransaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	TxnID    string             `bson:"txnid"`
	MihPayID string             `bson:"mihpayid"`
	ServiceID string            `bson:"serviceId"`
	UserID   string             `bson:"userId"`
	Amount   float64            `bson:"amount"`
	Status   string             `bson:"status"` // pending | processing | refunded | failed
	Reason   string             `bson:"reason"`
	CreatedAt time.Time         `bson:"createdAt"`
	UpdatedAt time.Time         `bson:"updatedAt"`
}
