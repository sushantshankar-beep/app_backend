package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type ServiceMaster struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	VehicleType string             `bson:"vehicleType" json:"vehicleType"`
	Logo        string             `bson:"logo" json:"logo"`
}
