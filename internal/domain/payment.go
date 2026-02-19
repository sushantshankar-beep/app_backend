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
	InvoiceGenerated bool `bson:"invoiceGenerated" json:"invoiceGenerated"`
	FailReason string `bson:"failReason,omitempty" json:"failReason,omitempty"`
	InvoiceGeneratedAt time.Time `bson:"invoiceGeneratedAt,omitempty"`
	TotalDiscount    float64 `bson:"totalDiscount" json:"totalDiscount"` 
    AmountPaidByUser float64 `bson:"amountPaidByUser" json:"amountPaidByUser"`
    PromoCode        string  `bson:"promoCode,omitempty" json:"promoCode,omitempty"` 
}

type PaymentWebhook struct {
	ID        string                 `bson:"_id,omitempty"`
	TxnID     string                 `bson:"txnid"`
	Payload   map[string]interface{} `bson:"payload"`
	CreatedAt time.Time              `bson:"createdAt"`
}
type PaymentStatus string

const (
	PaymentPending       PaymentStatus = "pending"
	PaymentPaid          PaymentStatus = "paid"
	PaymentFailed        PaymentStatus = "failed"
	PaymentFailedGrace   PaymentStatus = "failed_pending_release"
)
type PaymentFailReason string

const (
	FailUserCancelled PaymentFailReason = "user_cancelled"
	FailBankDecline   PaymentFailReason = "bank_declined"
	FailTimeout       PaymentFailReason = "timeout"
	FailGateway       PaymentFailReason = "gateway_error"
	FailUnknown       PaymentFailReason = "unknown"
)


