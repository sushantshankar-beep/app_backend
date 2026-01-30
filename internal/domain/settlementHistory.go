package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type SettlementRecord struct {
	ID                    primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ServiceID             primitive.ObjectID  `bson:"serviceId" json:"service_id"`
	PayoutID              primitive.ObjectID  `bson:"payoutId" json:"payout_id"`
	ProviderID            primitive.ObjectID  `bson:"providerId" json:"provider_id"`
	SettlementID          primitive.ObjectID  `bson:"settlementId" json:"settlement_id"`
	OriginalAmount        float64             `bson:"originalAmount" json:"original_amount"`
	PartialAmount         float64             `bson:"partialAmount" json:"partial_amount"`
	SettlementAmount      float64             `bson:"settlementAmount" json:"settlement_amount"`
	CommissionPercent     float64             `bson:"commissionPercent" json:"commission_percent"`
	CommissionAmount      float64             `bson:"commissionAmount" json:"commission_amount"`
	GSTPercent            float64             `bson:"gstPercent" json:"gst_percent"`
	GSTAmount             float64             `bson:"gstAmount" json:"gst_amount"`
	NetAmount             float64             `bson:"netAmount" json:"net_amount"`
	TDSPercent            float64             `bson:"tdsPercent" json:"tds_percent"`
    TDSAmount             float64             `bson:"tdsAmount" json:"tds_amount"`
	DeductionAmount       float64             `bson:"deductionAmount" json:"deduction_amount"`
	HasDeduction          bool                `bson:"hasDeduction" json:"has_deduction"`
	DeductionSettlementID *primitive.ObjectID `bson:"deductionSettlementId,omitempty" json:"deduction_settlement_id,omitempty"`
	DeductionPayoutID     *primitive.ObjectID `bson:"deductionPayoutId,omitempty" json:"deduction_payout_id,omitempty"`
	DeductionComplaintID  *primitive.ObjectID `bson:"deductionComplaintId,omitempty" json:"deduction_complaint_id,omitempty"`
	DeductionRemarks      string              `bson:"deductionRemarks,omitempty" json:"deduction_remarks,omitempty"`
	DeductionProcessedAt  *time.Time          `bson:"deductionProcessedAt,omitempty" json:"deduction_processed_at,omitempty"`
	SettlementType        string              `bson:"settlementType" json:"settlement_type"`
	ComplaintID           *primitive.ObjectID `bson:"complaintId,omitempty" json:"complaint_id,omitempty"`
	SettlementStatus      SettlementStatus    `bson:"settlementStatus" json:"settlement_status"`
	SettledAt             *time.Time          `bson:"settledAt,omitempty" json:"settled_at,omitempty"`
	CreatedAt             time.Time           `bson:"createdAt" json:"created_at"`
	UpdatedAt             time.Time           `bson:"updatedAt" json:"updated_at"`
	Remarks               string              `bson:"remarks,omitempty" json:"remarks,omitempty"`
}
