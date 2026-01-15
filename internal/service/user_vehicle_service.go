package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "fmt"
)

type UserVehicleService struct {
	vehicleRepo *repository.VehicleRepo
	userRepo    *repository.UserRepo
}

func NewUserVehicleService(
	vehicleRepo *repository.VehicleRepo,
	userRepo *repository.UserRepo,
) *UserVehicleService {
	return &UserVehicleService{
		vehicleRepo: vehicleRepo,
		userRepo:    userRepo,
	}
}

/*
GET vehicle by vehicleNumber (AUTO-FILL FLOW)
*/
func (s *UserVehicleService) GetVehicleByNumber(
	ctx context.Context,
	vehicleNumber string,
) (*domain.Vehicle, bool, error) {

	v, err := s.vehicleRepo.FindByNumber(ctx, vehicleNumber)
	if err != nil {
		return nil, false, err
	}
	if v == nil {
		return nil, false, nil
	}
	return v, true, nil
}

/*
SAVE vehicle and map to user
*/
func (s *UserVehicleService) SaveVehicleForUser(ctx context.Context, userID primitive.ObjectID, req map[string]string) (*domain.Vehicle, error) {
	v, err := s.vehicleRepo.FindByNumber(ctx, req["vehicleNumber"])
	if err != nil {
		return nil, err
	}

	if v == nil {
		v = &domain.Vehicle{
			VehicleNumber: req["vehicleNumber"],
			VehicleType:   req["vehicleType"],
			Brand:         req["brand"],
			Model:         req["model"],
			Source:        "manual",
			CreatedAt:     time.Now(),
		}

		if err := s.vehicleRepo.Create(ctx, v); err != nil {
			return nil, err
		}
	}
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.PrimaryVehicleID == nil {
		err = s.userRepo.SetPrimaryVehicle(ctx, userID, v.ID)
	} else {
		err = s.userRepo.AddFallbackVehicle(ctx, userID, v.ID)
	}

	if err != nil {
		return nil, err
	}

	return v, nil
}
