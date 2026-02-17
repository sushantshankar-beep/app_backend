package domain

type InvoiceJob struct {
	TxnID     string `json:"txnId"`
	UserID    string `json:"userId"`
	ServiceID string `json:"serviceId"`
}
