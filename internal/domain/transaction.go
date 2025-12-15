package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	TxnID         string             `bson:"txnid"`
	Amount        float64            `bson:"amount"`
	Status        string             `bson:"status"`
	UserID        primitive.ObjectID `bson:"userId"`
	ServiceID     primitive.ObjectID `bson:"serviceId"`
	MihPayID      string             `bson:"mihpayid,omitempty"`
	Method        string             `bson:"method,omitempty"`
	PaymentSource string             `bson:"paymentSource"`
	TxnResponse   any                `bson:"txnResponse,omitempty"`
	ErrorMessage  string             `bson:"errorMessage,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
}
