package domain

import "time"

type PaymentTransaction struct {
	ID            string    `bson:"_id,omitempty"`
	TxnID         string    `bson:"txnid"`
	Amount        float64   `bson:"amount"`
	Status        string    `bson:"status"`
	UserID        string    `bson:"userId"`
	ServiceID     string    `bson:"serviceId"`
	MihPayID      string    `bson:"mihpayid,omitempty"`
	Method        string    `bson:"method,omitempty"`
	PaymentSource string    `bson:"paymentSource"`
	ErrorMessage  string    `bson:"errorMessage,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}

type PaymentWebhook struct {
	ID        string                 `bson:"_id,omitempty"`
	TxnID     string                 `bson:"txnid"`
	Payload   map[string]interface{} `bson:"payload"`
	CreatedAt time.Time              `bson:"createdAt"`
}
