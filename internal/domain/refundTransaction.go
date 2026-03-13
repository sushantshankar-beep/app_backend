
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
	Timeline          RefundTimeline     `bson:"timeline"`
}

type PayURefundResponse struct {
	Msg       string `json:"msg"`
	ErrorCode int    `json:"error_code"`
	Status    int    `json:"status"`
	Mihpayid  int64  `json:"mihpayid"`
	RequestID string `json:"request_id"`
}


type RefundTimeline struct {
	Initiated     time.Time `bson:"initiated,omitempty" json:"initiated,omitempty"`
	UnderProcess  time.Time `bson:"underProcess,omitempty" json:"under_process,omitempty"`
	StatusChecked time.Time `bson:"statusChecked,omitempty" json:"status_checked,omitempty"`
	Completed     time.Time `bson:"completed,omitempty" json:"completed,omitempty"`
	Failed        time.Time `bson:"failed,omitempty" json:"failed,omitempty"`
}