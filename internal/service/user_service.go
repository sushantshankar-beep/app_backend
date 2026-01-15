package service

import (
	"context"
	"time"
	"errors"
	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/worker"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService struct {
	users   ports.UserRepository
	otp     ports.OTPStore
	token   ports.TokenService
	queue   *worker.OTPQueue
	amcRepo *repository.AMCRepo
}

func NewUserService(users ports.UserRepository, otp ports.OTPStore, token ports.TokenService, q *worker.OTPQueue) *UserService {
	return &UserService{users: users, otp: otp, token: token, queue: q}
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
func (s *UserService) GetProfile(ctx context.Context, userObjID primitive.ObjectID) (*domain.User, error) {
	fmt.Println("this is user id in get profile", userObjID)
	return s.users.GetByID(ctx, userObjID)
}

func (s *UserService) VerifyOTP(ctx context.Context, phone, code string) (string, bool, error) {
	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return "", false, domain.ErrOTPInvalid
	}

	if time.Now().After(otp.ExpiresAt) {
		return "", false, domain.ErrOTPExpired
	}
	_ = s.otp.Delete(ctx, phone)

	u, err := s.users.FindByPhone(ctx, phone)
	isNew := false
	// HERE IF OTP IS VERIFIED THEN WE SAVE IT AS SERVICE NO EXTRA GENERATION LOGIC
	if err == domain.ErrNotFound {
		isNew = true

		u = &domain.User{
			Phone:      phone,
			ServiceOTP: code,
			CreatedAt:  time.Now(),
		}

		if err := s.users.Create(ctx, u); err != nil {
			return "", false, err
		}

	} else if err != nil {
		return "", false, err
	}
	fmt.Println(u.ID)
	token, err := s.token.GenerateUserToken(u.ID)
	return token, isNew, err
}
func (s *UserService) CreateOrUpdateProfile(ctx context.Context, userID domain.UserID, req map[string]any) (*domain.User, string, error) {

	objID, err := primitive.ObjectIDFromHex(string(userID))
	fmt.Println(objID, err)
	if err != nil {
		return nil, "", errors.New("invalid user id")
	}

	user, err := s.users.GetByID(ctx, objID)
	if err != nil {
		return nil, "", err
	}

	isCreate := user.Name == "" &&
		user.Email == "" &&
		user.SelectedCity == ""

	update := bson.M{}
	setString(update, "name", req["name"])
	setString(update, "email", req["email"])
	setString(update, "fcmToken", req["fcmToken"])
	setString(update, "selectedCity", req["selectedCity"])
	setString(update, "appStateStatus", req["appStateStatus"])
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
