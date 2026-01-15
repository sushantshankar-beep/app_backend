package events

type PaymentEvent struct {
	TxnID     string `json:"txnid"`
	ServiceID string `json:"serviceId"`
	Status    string `json:"status"`
}

type ServiceEvent struct {
	ServiceID string `json:"serviceId"`
	Status    string `json:"status"`
}


type ProviderEvent struct {
	ServiceID  string `json:"serviceId"`
	ProviderID string `json:"providerId"`
	Action     string `json:"action"` // assign | release
}
