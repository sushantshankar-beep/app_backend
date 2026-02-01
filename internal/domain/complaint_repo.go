package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ComplaintSide struct {
	Problem  string    `bson:"problem" json:"problem"`
	Photos   []string  `bson:"photos" json:"photos"`
	RaisedAt time.Time `bson:"raisedAt" json:"raisedAt"`
}

type Complaint struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AcceptedService primitive.ObjectID `bson:"acceptedService" json:"acceptedService"`
	ComplaintNumber string             `bson:"complaintNumber" json:"complaintNumber"`

	ProviderID        primitive.ObjectID   `bson:"providerId" json:"providerId"`
	UserID            primitive.ObjectID   `bson:"userId" json:"userId"`
	Assessment        *ComplaintAssessment `bson:"assessment,omitempty" json:"assessment,omitempty"`
	UserComplaint     *ComplaintSide       `bson:"userComplaint,omitempty" json:"userComplaint,omitempty"`
	ProviderComplaint *ComplaintSide       `bson:"providerComplaint,omitempty" json:"providerComplaint,omitempty"`
	ServiceNumber     string               `bson:"serviceNumber" json:"serviceNumber"`
	Status            string               `bson:"status" json:"status"`
	Timeline          map[string]time.Time `bson:"timeline" json:"timeline"`
	CreatedAt         time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type ComplaintAssessment struct {
	FaultParty        FaultParty `bson:"faultParty" json:"fault_party"`
	RefundToUser      RefundType `bson:"refundToUser" json:"refund_to_user"`
	RefundAmount      float64    `bson:"refundAmount,omitempty" json:"refund_amount,omitempty"`
	PayoutToProvider  PayoutType `bson:"payoutToProvider" json:"payout_to_provider"`
	PayoutAmount      float64    `bson:"payoutAmount,omitempty" json:"payout_amount,omitempty"`
	RemarkForUser     string     `bson:"remarkForUser,omitempty" json:"remarkForUser,omitempty"`
	RemarkForProvider string     `bson:"remarkForProvider,omitempty" json:"remarkForProvider,omitempty"`
	AssessedBy        string     `bson:"assessedBy" json:"assessed_by"`
	AssessedAt        time.Time  `bson:"assessedAt" json:"assessed_at"`
}

type FaultParty string

const (
	FaultPartyUser     FaultParty = "user"
	FaultPartyProvider FaultParty = "provider"
	FaultPartyBoth     FaultParty = "both"
	FaultPartySystem   FaultParty = "system"
)

type RefundType string

const (
	RefundTypeFull    RefundType = "Full Refund"
	RefundTypePartial RefundType = "Partial Refund"
	RefundTypeNone    RefundType = "No Refund"
)

type PayoutType string

const (
	PayoutTypeFull    PayoutType = "Full Payout"
	PayoutTypePartial PayoutType = "Partial Payout"
	PayoutTypeNone    PayoutType = "No Payout"
)
