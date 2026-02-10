package service

import (
	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/worker"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserService struct {
	users   ports.UserRepository
	otp     ports.OTPStore
	token   ports.TokenService
	queue   *worker.OTPQueue
	counter *repository.CounterRepo
	amcRepo *repository.AMCRepo
	vehicleRepo *repository.VehicleRepo
	db *mongo.Database
}
var mongoDB *mongo.Database
func NewUserService(users ports.UserRepository, otp ports.OTPStore, token ports.TokenService, q *worker.OTPQueue, counter *repository.CounterRepo,vehicleRepo *repository.VehicleRepo,db *mongo.Database) *UserService {
	return &UserService{users: users, otp: otp, token: token, queue: q, counter: counter, vehicleRepo: vehicleRepo,db:db}
}

func GenerateOTP() string {
	b := make([]byte, 8)
	rand.Read(b)
	n := binary.BigEndian.Uint64(b)
	otp := n % 10000
	return fmt.Sprintf("%04d", otp)
}

func (s *UserService) SendOTP(ctx context.Context, phone string) error {
	code := GenerateOTP()
	otp := &domain.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.otp.Save(ctx, otp); err != nil {
		return err
	}

	s.queue.Enqueue(worker.OTPJob{Phone: phone, Msg: code, Type: "user"})
	return nil
}
func (s *UserService) GetProfile(ctx context.Context, userObjID primitive.ObjectID) (map[string]interface{}, error) {
	fmt.Println("this is user id in get profile", userObjID)
	user, err := s.users.GetByID(ctx, userObjID)
	if err != nil {
		return nil, err
	}
	vehicleCount := int64(0)
	if user.PrimaryVehicleID != nil {
		vehicleCount++
	}
	if user.FallbackVehicleIDs != nil {
		vehicleCount += int64(len(user.FallbackVehicleIDs))
	}
	result := map[string]interface{}{
		"id":                  user.ID,
		"userCode":            user.UserCode,
		"phone":               user.Phone,
		"name":                user.Name,
		"createdAt":           user.CreatedAt,
		"updatedAt":           user.UpdatedAt,
		"email":               user.Email,
		"image_url":           user.ImageUrl,
		"referralCode":        user.ReferralCode,
		"isActive":            user.IsActive,
		"fcmToken":            user.FcmToken,
		"appStateStatus":      user.AppStateStatus,
		"isProfileComplete":   user.IsProfileComplete,
		"selectedCity":        user.SelectedCity,
		"amcPurchased":        user.AmcPurchased,
		"complaintsSubmitted": user.ComplaintsSubmitted,
		"service_otp":         user.ServiceOTP,
		"totalExpense":        user.TotalExpense,
		"rating":              user.Rating,
		"primaryVehicleId":    user.PrimaryVehicleID,
		"fallbackVehicleIds":  user.FallbackVehicleIDs,
		"vehicleCount":        vehicleCount,
	}
	if user.VehicleID != nil {
		result["vehicleId"] = user.VehicleID
	}
	return result, nil
}
func (s *UserService) VerifyOTP(
	ctx context.Context,
	phone, code string,
) (map[string]any, error) {

	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return nil, domain.ErrOTPInvalid
	}

	if time.Now().After(otp.ExpiresAt) {
		return nil, domain.ErrOTPExpired
	}
	_ = s.otp.Delete(ctx, phone)
	
	u, err := s.users.FindByPhone(ctx, phone)
	isNew := false

	if err == domain.ErrNotFound || (err == nil && u.IsActive == domain.USER_DELETED) {
		isNew = true

		seq, _ := s.counter.Next(ctx, "user")
		year := time.Now().Year()

		u = &domain.User{
			Phone:             phone,
			UserCode:          fmt.Sprintf("VHCR%d%05d", year, seq),
			IsProfileComplete: false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			ServiceOTP:        code,
		}

		if err := s.users.Create(ctx, u); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	token, err := s.token.GenerateUserToken(u.ID)
	if err != nil {
		return nil, err
	}

	nextScreen := "HOME"
	if !u.IsProfileComplete {
		nextScreen = "PROFILE"
	}

	return map[string]any{
		"token":           token,
		"isNew":           isNew,
		"profileComplete": u.IsProfileComplete,
		"userCode":        u.UserCode,
		"nextScreen":      nextScreen,
	}, nil
}

func (s *UserService) CreateOrUpdateProfile(ctx context.Context, userID domain.UserID, req map[string]any) (*domain.User, string, error) {

	objID, err := primitive.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, "", errors.New("invalid user id")
	}
	
	user, err := s.users.GetByID(ctx, objID)
	if err != nil {
		return nil, "", err
	}

	isCreate := user.Name == "" && user.Email == "" && user.SelectedCity == ""
	
	update := bson.M{}
	setString(update, "name", req["name"])
	setString(update, "email", req["email"])
	setString(update, "fcmToken", req["fcmToken"])
	setString(update, "selectedCity", req["selectedCity"])
	setString(update, "appStateStatus", req["appStateStatus"])
	setString(update, "image_url", req["imageUrl"])
	
	if val, ok := req["isActive"]; ok {
		setString(update, "isActive", val)
	}

	if update["name"] != nil && update["email"] != nil {
		update["isProfileComplete"] = true
	}
	
	update["updatedAt"] = time.Now()
	
	if len(update) == 1 {
		return user, "unchanged", nil
	}
	
	updatedUser, err := s.users.UpdateByID(ctx, objID, update)
	if err != nil {
		return nil, "", err
	}
	
	if isCreate {
		return updatedUser, "created", nil
	}
	return updatedUser, "updated", nil
}

func setString(update bson.M, key string, v any) {
	if s, ok := v.(string); ok && s != "" {
		update[key] = s
	}
}
func (s *UserService) GetActiveAMCByUser(ctx context.Context, userID primitive.ObjectID) (*domain.AMC, error) {
	return s.amcRepo.FindActiveByUser(ctx, userID)
}


func (s *UserService) Logout(
	ctx context.Context,
	userID domain.UserID,
	token string, // ignored
) error {

	filter := bson.M{
		"ownerId":   string(userID),
		"ownerType": "user",
	}
	fmt.Println("THIS IS USER ID",userID)
	fmt.Println("THIS IS TOKEN",token)
	fmt.Println(filter)

	opts := options.FindOneAndDelete().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	res := s.db.
	Collection("device_tokens").
	FindOneAndDelete(ctx, filter, opts)
	fmt.Println(res)

	if res.Err() == mongo.ErrNoDocuments {
		return errors.New("no active session found")
	}

	if res.Err() != nil {
		return res.Err()
	}

	return nil
}




func (s *UserService) DeleteUser(ctx context.Context, userID primitive.ObjectID) error {
	_ , err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"isActive":  domain.USER_DELETED,
		"isNew": true,
		"isProfileComplete": false,
		"updatedAt": time.Now(),
	}

	_, err = s.users.UpdateByID(ctx, userID, update)
	return err
}

func (s *UserService) GetUserVehicleCount(ctx context.Context, userObjID primitive.ObjectID) (int64, error) {
	return s.vehicleRepo.CountByUserID(ctx, userObjID)
}