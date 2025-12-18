package domain

type RefundJob struct {
	MihPayID string  `json:"mihpayid"`
	Amount   float64 `json:"amount"`
	Retries  int     `json:"retries"`
}
