package dto

import "app_backend/internal/domain"

type PromoCodeListResponse struct {
	ID            string              `json:"id"`
	Code          string              `json:"code"`
	Title         string              `json:"title"`
	ServiceType   string              `json:"service_type"`
	ValidityStart string              `json:"startAt"`
	ValidityEnd   string              `json:"endAt"`
	Status        domain.PromoStatus  `json:"status"`
	DiscountType  domain.DiscountType `json:"discount_type"`
	Value         float64             `json:"value"`
	MaxDiscount   float64             `json:"max_discount"`
	MinOrderValue float64             `json:"min_order_value"`
	ImageURL      string              `json:"imageUrl"`
	Description   string              `json:"description"`
	CreatedBy     string              `json:"created_by"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at"`
}
