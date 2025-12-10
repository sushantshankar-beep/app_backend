package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/worker"
)

type UserService struct {
	users ports.UserRepository
	otp   ports.OTPStore
	token ports.TokenService
	queue *worker.OTPQueue
}

func NewUserService(users ports.UserRepository, otp ports.OTPStore, token ports.TokenService, q *worker.OTPQueue) *UserService {
	return &UserService{users: users, otp: otp, token: token, queue: q}
}

func (s *UserService) SendOTP(ctx context.Context, phone string) error {
	code := "1234"

	otp := &domain.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.otp.Save(ctx, otp); err != nil {
		return err
	}

	s.queue.Enqueue(worker.OTPJob{Phone: phone, Msg: "Your OTP is " + code})
	return nil
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

	if err == domain.ErrNotFound {
		isNew = true
		u = &domain.User{
			Phone:     phone,
			CreatedAt: time.Now(),
		}
		if err := s.users.Create(ctx, u); err != nil {
			return "", false, err
		}
	} else if err != nil {
		return "", false, err
	}

	token, err := s.token.GenerateUserToken(u.ID)
	return token, isNew, err
}
