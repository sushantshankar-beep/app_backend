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
	ComplaintNumber    string             `bson:"complaintNumber" json:"complaintNumber"`

	ProviderID primitive.ObjectID `bson:"providerId" json:"providerId"`
	UserID     primitive.ObjectID `bson:"userId" json:"userId"`

	UserComplaint     *ComplaintSide `bson:"userComplaint,omitempty" json:"userComplaint,omitempty"`
	ProviderComplaint *ComplaintSide `bson:"providerComplaint,omitempty" json:"providerComplaint,omitempty"`
     ServiceNumber string `bson:"serviceNumber" json:"serviceNumber"`
	Status    string               `bson:"status" json:"status"`
	Timeline  map[string]time.Time `bson:"timeline" json:"timeline"`
	CreatedAt time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time            `bson:"updatedAt" json:"updatedAt"`
}
