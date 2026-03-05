package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DiscountScope string

const (
	DiscountScopePlatformWide   DiscountScope = "platform_wide"
	DiscountScopeNewUsers       DiscountScope = "new_users"
	DiscountScopeSelectedUsers  DiscountScope = "selected_users"
	DiscountScopeSpecificPlans  DiscountScope = "specific_plans"
)

type DiscountApplicableOn string

const (
	DiscountApplicableServices DiscountApplicableOn = "Services"
	DiscountApplicableAMC      DiscountApplicableOn = "AMC Plans"
)

type DiscountStatus string

const (
	DiscountStatusDraft     DiscountStatus = "draft"
	DiscountStatusActive    DiscountStatus = "active"
	DiscountStatusInActive    DiscountStatus = "inActive"
	DiscountStatusScheduled DiscountStatus = "scheduled"
	DiscountStatusExpired   DiscountStatus = "expired"
	DiscountStatusPaused    DiscountStatus = "paused"
)

type Discount struct {
	ID                    primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    Code                  string                 `bson:"code" json:"code"`
	Name                  string                 `bson:"name" json:"name"`
	Description           string                 `bson:"description" json:"description"`
	Type                  DiscountType           `bson:"type" json:"type"`
	Value                 float64                `bson:"value" json:"value"`
	MaxDiscount           float64                `bson:"maxDiscount" json:"max_discount"`
	Scope                 DiscountScope          `bson:"scope" json:"scope"`
	ApplicableOn          []DiscountApplicableOn `bson:"applicableOn" json:"applicable_on"`
	Zones                 []string               `bson:"zones" json:"zones"`
	PaymentMethods        []string               `bson:"paymentMethods" json:"payment_methods"`
	UserEligibility       UserEligibility        `bson:"userEligibility" json:"user_eligibility"`
	AllowStackingWithPromo bool                  `bson:"allowStackingWithPromo" json:"allow_stacking_with_promo"`
	Status                DiscountStatus         `bson:"status" json:"status"`
	TotalSavings          float64                `bson:"totalSavings" json:"total_savings"`
	TotalOrders           int                    `bson:"totalOrders" json:"total_orders"`
	CreatedBy             string                 `bson:"createdBy" json:"created_by"`
	StartAt               time.Time              `bson:"startAt" json:"start_at"`
	EndAt                 *time.Time             `bson:"endAt,omitempty" json:"end_at,omitempty"`
	CreatedAt             time.Time              `bson:"createdAt" json:"created_at"`
	UpdatedAt             time.Time              `bson:"updatedAt" json:"updated_at"`
	VehicleType           string                 `bson:"vehicleType" json:"vehicle_type"` 
}

func IsValidDiscountScope(s DiscountScope) bool {
	return s == DiscountScopePlatformWide ||
		s == DiscountScopeNewUsers ||
		s == DiscountScopeSelectedUsers ||
		s == DiscountScopeSpecificPlans
}

func IsValidDiscountStatus(s DiscountStatus) bool {
	return s == DiscountStatusDraft ||
		s == DiscountStatusActive ||
		s == DiscountStatusScheduled ||
		s == DiscountStatusExpired ||
		s == DiscountStatusPaused ||
		s == DiscountStatusInActive
}

func IsValidDiscountApplicableOn(a DiscountApplicableOn) bool {
	return a == DiscountApplicableServices || a == DiscountApplicableAMC
}

type AppliedDiscountSummary struct {
	DiscountID  string  `bson:"discountId,omitempty" json:"discountId,omitempty"`
	Code        string  `bson:"code,omitempty" json:"code,omitempty"`
	DiscountAmt float64 `bson:"discountAmt" json:"discountAmt"`
}
