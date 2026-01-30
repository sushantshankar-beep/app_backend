
package domain

type RefundJob struct {
	ServiceID string  `json:"serviceId"`
	MihPayID  string  `json:"mihPayId"`
	Amount    float64 `json:"amount"`
	Retries   int     `json:"retries"`
}