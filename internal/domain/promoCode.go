package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DiscountType string

const (
	DiscountPercent DiscountType = "percent"
	DiscountFlat    DiscountType = "flat"
)

type PromoServiceType string

const (
	PromoServiceAll      PromoServiceType = "All Services"
	PromoServiceMechanic PromoServiceType = "Mechanic"
	PromoServiceAMC      PromoServiceType = "AMC"
	PromoServiceTowAway  PromoServiceType = "Tow Away"
	PromoServicePetrol   PromoServiceType = "Petrol"
)

type PromoStatus string

const (
	PromoStatusDraft     PromoStatus = "draft"
	PromoStatusActive    PromoStatus = "active"
	PromoStatusScheduled PromoStatus = "scheduled"
	PromoStatusExpired   PromoStatus = "expired"
	PromoStatusInactive  PromoStatus = "inactive"
)

type UserEligibility string

const (
	UserEligibilityAll      UserEligibility = "all_users"
	UserEligibilityNew      UserEligibility = "new_users"
	UserEligibilityExisting UserEligibility = "existing_users"
)

type PromoCode struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code                string             `bson:"code" json:"code"`
	Title               string             `bson:"title" json:"title"`
	Description         string             `bson:"description" json:"description"`
	DiscountType        DiscountType       `bson:"discountType" json:"discount_type"`
	Value               float64            `bson:"value" json:"value"`
	MaxDiscount         float64            `bson:"maxDiscount" json:"max_discount"`
	MinOrderValue       float64            `bson:"minOrderValue" json:"min_order_value"`
	PerUserLimit        int                `bson:"perUserLimit" json:"per_user_limit"`
	GlobalRedemptionCap int                `bson:"globalRedemptionCap" json:"global_redemption_cap"`
	ServiceTypes        []PromoServiceType `bson:"serviceTypes" json:"service_types"`
	Zones               []string           `bson:"zones" json:"zones"`
	PaymentMethods      []string           `bson:"paymentMethods" json:"payment_methods"`
	UserEligibility     UserEligibility    `bson:"userEligibility" json:"user_eligibility"`
	AllowStacking       bool               `bson:"allowStacking" json:"allow_stacking"`
	Status              PromoStatus        `bson:"status" json:"status"`
	UsageCount          int                `bson:"usageCount" json:"usage_count"`
	TotalDiscount       float64            `bson:"totalDiscount" json:"total_discount"`
	CreatedBy           string             `bson:"createdBy" json:"created_by"`
	StartAt             time.Time          `bson:"startAt" json:"start_at"`
	EndAt               *time.Time         `bson:"endAt,omitempty" json:"end_at,omitempty"`
	CreatedAt           time.Time          `bson:"createdAt" json:"created_at"`
	UpdatedAt           time.Time          `bson:"updatedAt" json:"updated_at"`
	VehicleType         string             `bson:"vehicleType" json:"vehicle_type"`

}

type PromoActivityLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PromoID   primitive.ObjectID `bson:"promoId" json:"promo_id"`
	Action    string             `bson:"action" json:"action"`
	CreatedBy string             `bson:"createdBy" json:"created_by"`
	CreatedAt time.Time          `bson:"createdAt" json:"created_at"`
}

func IsValidDiscountType(dt DiscountType) bool {
	return dt == DiscountPercent || dt == DiscountFlat
}

func IsValidPromoStatus(s PromoStatus) bool {
	return s == PromoStatusDraft ||
		s == PromoStatusActive ||
		s == PromoStatusScheduled ||
		s == PromoStatusExpired ||
		s == PromoStatusInactive
}

func IsValidUserEligibility(u UserEligibility) bool {
	return u == UserEligibilityAll || u == UserEligibilityNew || u == UserEligibilityExisting
}

func IsValidPromoServiceType(s PromoServiceType) bool {
	return s == PromoServiceAll ||
		s == PromoServiceMechanic ||
		s == PromoServiceAMC ||
		s == PromoServiceTowAway ||
		s == PromoServicePetrol
}

type AppliedPromoSummary struct {
	PromoID     string  `bson:"promoId,omitempty" json:"promoId,omitempty"`
	Code        string  `bson:"code,omitempty" json:"code,omitempty"`
	DiscountAmt float64 `bson:"discountAmt" json:"discountAmt"`
}

type CouponApplyResult struct {
    ServiceAmount    float64                 `bson:"serviceAmount"    json:"serviceAmount"`
    DiscountedAmount float64                 `bson:"discountedAmount" json:"discountedAmount"`
    TotalDiscount    float64                 `bson:"totalDiscount"    json:"totalDiscount"`
    AppliedPromo     *AppliedPromoSummary    `bson:"appliedPromo,omitempty"    json:"appliedPromo,omitempty"`
    AppliedDiscount  *AppliedDiscountSummary `bson:"appliedDiscount,omitempty" json:"appliedDiscount,omitempty"`
}
