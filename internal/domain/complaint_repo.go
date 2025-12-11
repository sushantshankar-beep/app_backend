package domain

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Complaint struct {
    ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    AcceptedService primitive.ObjectID `bson:"acceptedService" json:"acceptedService"`
    AcceptedServiceId int64            `bson:"acceptedServiceId" json:"acceptedServiceId"`

    ProviderID      primitive.ObjectID `bson:"providerId" json:"providerId"`
    UserID          primitive.ObjectID `bson:"userId" json:"userId"`

    RaisedBy        string             `bson:"raisedBy" json:"raisedBy"` // "User" or "Provider"
    Problem         string             `bson:"problem" json:"problem"`
    Photos          []string           `bson:"photos" json:"photos"`

    Status          string             `bson:"status" json:"status"` // "initiated"
    Timeline        map[string]time.Time `bson:"timeline" json:"timeline"`

    CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
    UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`
}
