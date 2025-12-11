package validation

import (
	"errors"
	"strings"
	"time"
)

type ServiceRequestRequest struct {
	VehicleNumber string     `json:"vehicleNumber" binding:"required"`
	VehicleType   string     `json:"vehicleType"`
	Brand         string     `json:"brand" binding:"required"`
	Model         string     `json:"model" binding:"required"`
	Year          *int       `json:"year"`
	FuelType      string     `json:"fuelType" binding:"required"`
	ServiceType   string     `json:"serviceType"`
	Problems      []string   `json:"problems"`
	Description   string     `json:"description"`
	Address       string     `json:"address"`
	ScheduledDate *time.Time `json:"scheduledDate"`
	RadiusUnit    int        `json:"radiusUnit"`
	RequestID     string     `json:"requestId"`
}

func (r *ServiceRequestRequest) Validate() error {
	if strings.TrimSpace(r.VehicleNumber) == "" {
		return errors.New("vehicleNumber is required")
	}

	if strings.TrimSpace(r.Brand) == "" {
		return errors.New("brand is required")
	}

	if strings.TrimSpace(r.Model) == "" {
		return errors.New("model is required")
	}

	if strings.TrimSpace(r.FuelType) == "" {
		return errors.New("fuelType is required")
	}

	if r.Year != nil && (*r.Year < 1900 || *r.Year > time.Now().Year()+1) {
		return errors.New("year must be between 1900 and current year")
	}

	return nil
}