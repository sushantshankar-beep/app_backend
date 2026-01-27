package service

import (
	"context"
	"time"
    "errors"
 	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"app_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserVehicleService struct {
	carBrandModelRepo *repository.CarBrandModelRepo
	bikeBrandModelRepo *repository.BikeBrandModelRepo
	vehicleRepo *repository.VehicleRepo
	userRepo    *repository.UserRepo

}

func NewUserVehicleService(
	carBrandModelRepo *repository.CarBrandModelRepo,
	bikeBrandModelRepo *repository.BikeBrandModelRepo,
	vehicleRepo *repository.VehicleRepo,
	userRepo *repository.UserRepo,
) *UserVehicleService {
	return &UserVehicleService{
		carBrandModelRepo: carBrandModelRepo,
		bikeBrandModelRepo: bikeBrandModelRepo,
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
			ModelYear:        req["modelYear"],
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

func (s *UserVehicleService) GetVehicleData( ctx context.Context, vehicleType string, make string, model string ) ([]domain.Brand, error) {

	if vehicleType != "" && vehicleType != "car" && vehicleType != "bike" {
		return nil, errors.New("vehicleType must be car or bike")
	}

	if vehicleType == "car" {
		return s.getCarData(ctx, make, model)
	} else if vehicleType == "bike" {
		return s.getBikeData(ctx, make, model)
	}

	return nil, errors.New("vehicleType is required")
}

func (s *UserVehicleService) getCarData(ctx context.Context, make string, model string) ([]domain.Brand, error) {
	filter := bson.M{}
	if make != "" {
		filter["make"] = make
	}
	if model != "" {
		filter["models.model"] = model
	}
	return s.carBrandModelRepo.GetAll(ctx, filter)
}

func (s *UserVehicleService) getBikeData(ctx context.Context, make string, model string) ([]domain.Brand, error) {
	filter := bson.M{}
	if make != "" {
		filter["make"] = make
	}
	if model != "" {
		filter["models.model"] = model
	}
	return s.bikeBrandModelRepo.GetAll(ctx, filter)
}
type UserVehiclesResponse struct {
	Primary   *domain.Vehicle   `json:"primaryVehicle"`
	Fallbacks []domain.Vehicle `json:"fallbackVehicles"`
}

func (s *UserVehicleService) GetMyVehicles(
	ctx context.Context,
	userID primitive.ObjectID,
) (*UserVehiclesResponse, error) {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &UserVehiclesResponse{}

	if user.PrimaryVehicleID != nil {

		v, err := s.vehicleRepo.FindByIDs(
			ctx,
			[]primitive.ObjectID{*user.PrimaryVehicleID},
		)

		if err == nil && len(v) > 0 {
			resp.Primary = &v[0]
		}
	}

	if len(user.FallbackVehicleIDs) > 0 {

		vehicles, err := s.vehicleRepo.FindByIDs(
			ctx,
			user.FallbackVehicleIDs,
		)

		if err != nil {
			return nil, err
		}

		resp.Fallbacks = vehicles
	}

	return resp, nil
}

