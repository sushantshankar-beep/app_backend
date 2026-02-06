package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Vehicle struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VehicleNumber string             `bson:"vehicleNumber" json:"vehicleNumber"`
	VehicleType   string             `bson:"vehicleType" json:"vehicleType"`
	Brand         string             `bson:"brand" json:"brand"`
	Model         string             `bson:"model" json:"model"`
	ModelYear    string          	 `bson:"modelYear" json:"modelYear"`
	FuelType     string               `bson:"fuelType" json:"fuelType"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
}

type UserVehicle struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"userId" json:"userId"`
	VehicleID 	   primitive.ObjectID		 `bson:"vehicle_id" json:"vehicle_id"`

	VehicleNumber string             `bson:"vehicleNumber" json:"vehicleNumber"`
	VehicleType   string             `bson:"vehicleType" json:"vehicleType"`
	Brand         string             `bson:"brand" json:"brand"`
	Model         string             `bson:"model" json:"model"`
	ModelYear    string          	 `bson:"modelYear" json:"modelYear"`
	FuelType     string             `bson:"fuelType" json:"fuelType"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}
