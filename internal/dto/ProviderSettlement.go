package dto

type ProviderSettlementItem struct {
	ID            string  `json:"id"`            
	ProviderID    string  `json:"provider"`
	ServiceNumber string  `json:"serviceNumber"`
	Amount        float64 `json:"amount"`       
	CreatedAt     string  `json:"createdAt"`
}

type ProviderSettlementResponse struct {
	Total       float64                 `json:"total"`
	Settlements []ProviderSettlementItem `json:"settlements"`
}
