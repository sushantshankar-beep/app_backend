package dto

import "time"

type UserExpenseDTO struct {
	ServiceID     string    `json:"serviceId"`
	ServiceNumber string    `json:"serviceNumber"`
	ServiceType   string    `json:"serviceType"`
	ProviderName  string    `json:"providerName"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"paymentMethod"`
	CreatedAt     time.Time `json:"createdAt"`
	VehicleType   string    `json:"vehicleType"`
	VehicleNumber string    `json:"vehicleNumber"`
	Issues        []string  `json:"issues,omitempty"`
}
