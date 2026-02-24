package dto

import "app_backend/internal/domain"

type PromoCodeListResponse struct {
	ID            string   `json:"id"`
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	ServiceType   string   `json:"service_type"`
	ValidityStart string   `json:"startAt"`
	ValidityEnd   string  `json:"endAt"`
    Status        domain.PromoStatus   `json:"status"`
	CreatedBy     string   `json:"created_by"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}
