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
			FuelType:       req["fuelType"],
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
		if err := s.userRepo.SetPrimaryVehicle(ctx, userID, v.ID); err != nil {
			return nil, err
		}
		return v, nil
	}

	if u.PrimaryVehicleID.Hex() == v.ID.Hex() {
		return v, nil
	}

	for _, fid := range u.FallbackVehicleIDs {
		if fid.Hex() == v.ID.Hex() {
			return v, nil
		}
	}

	if err := s.userRepo.AddFallbackVehicle(ctx, userID, v.ID); err != nil {
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

func (s *UserVehicleService) UpdateVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleID primitive.ObjectID,
	req map[string]string,
) (*domain.Vehicle, error) {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	isOwned := false
	if user.PrimaryVehicleID != nil && *user.PrimaryVehicleID == vehicleID {
		isOwned = true
	}
	for _, fid := range user.FallbackVehicleIDs {
		if fid == vehicleID {
			isOwned = true
			break
		}
	}

	if !isOwned {
		return nil, errors.New("vehicle not found or unauthorized")
	}

	vehicle, err := s.vehicleRepo.FindByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if vehicle == nil {
		return nil, errors.New("vehicle not found")
	}

	updateData := bson.M{}
	if val, ok := req["vehicleNumber"]; ok && val != "" {
		updateData["vehicleNumber"] = val
	}
	if val, ok := req["vehicleType"]; ok && val != "" {
		updateData["vehicleType"] = val
	}
	if val, ok := req["brand"]; ok && val != "" {
		updateData["brand"] = val
	}
	if val, ok := req["model"]; ok && val != "" {
		updateData["model"] = val
	}
	if val, ok := req["modelYear"]; ok && val != "" {
		updateData["modelYear"] = val
	}
	if val,ok := req["fuelType"]; ok && val != "" {
		updateData["fuelType"] = val
	}

	if len(updateData) == 0 {
		return nil, errors.New("no fields to update")
	}

	err = s.vehicleRepo.Update(ctx, vehicleID, updateData)
	if err != nil {
		return nil, err
	}

	updatedVehicle, err := s.vehicleRepo.FindByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	return updatedVehicle, nil
}

func (s *UserVehicleService) DeleteVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleID primitive.ObjectID,
) error {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.PrimaryVehicleID != nil && *user.PrimaryVehicleID == vehicleID {
		
		if len(user.FallbackVehicleIDs) > 0 {
			newPrimaryID := user.FallbackVehicleIDs[0]
			newFallbacks := user.FallbackVehicleIDs[1:]

			err = s.userRepo.SetPrimaryVehicle(ctx, userID, newPrimaryID)
			if err != nil {
				return err
			}

			err = s.userRepo.SetFallbackVehicles(ctx, userID, newFallbacks)
			if err != nil {
				return err
			}

			return nil
		}

		err = s.userRepo.ClearPrimaryVehicle(ctx, userID)
		if err != nil {
			return err
		}

		return nil
	}

	found := false
	newFallbacks := []primitive.ObjectID{}
	for _, fid := range user.FallbackVehicleIDs {
		if fid == vehicleID {
			found = true
			continue
		}
		newFallbacks = append(newFallbacks, fid)
	}

	if !found {
		return errors.New("vehicle not found or unauthorized")
	}

	err = s.userRepo.SetFallbackVehicles(ctx, userID, newFallbacks)
	if err != nil {
		return err
	}

	return nil
}