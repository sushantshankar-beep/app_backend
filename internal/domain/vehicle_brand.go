package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type VehicleBrand struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VehicleType string             `bson:"vehicleType" json:"vehicleType"`
	BrandName   string             `bson:"brandName" json:"brandName"`
}
