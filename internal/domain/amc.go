package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AMC struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	VehicleNumber  string             `bson:"vehicleNumber" json:"vehicleNumber"`

	PlanName       string             `bson:"planName" json:"planName"`
	ValidServices  []string           `bson:"validServices" json:"validServices"`

	StartDate      time.Time          `bson:"startDate" json:"startDate"`
	EndDate        time.Time          `bson:"endDate" json:"endDate"`

	IsActive       bool               `bson:"isActive" json:"isActive"`

	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}
type ValidServices struct{
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name    string             `bson:"name" json:"name"`
	
	
}
