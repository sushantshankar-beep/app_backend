package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/worker"
)

type ProviderService struct {
	repo  ports.ProviderRepository
	otp   ports.OTPStore
	token ports.TokenService
	queue *worker.OTPQueue
}

func NewProviderService(repo ports.ProviderRepository, otp ports.OTPStore, token ports.TokenService, q *worker.OTPQueue) *ProviderService {
	return &ProviderService{repo: repo, otp: otp, token: token, queue: q}
}

func (s *ProviderService) SendOTP(ctx context.Context, phone string) error {
	code := "1234"

	otp := &domain.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.otp.Save(ctx, otp); err != nil {
		return err
	}

	s.queue.Enqueue(worker.OTPJob{Phone: phone, Msg: "Your provider OTP is " + code})
	return nil
}

func (s *ProviderService) VerifyOTP(ctx context.Context, phone, code string) (string, bool, error) {
	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return "", false, domain.ErrOTPInvalid
	}
	if time.Now().After(otp.ExpiresAt) {
		return "", false, domain.ErrOTPExpired
	}

	_ = s.otp.Delete(ctx, phone)

	p, err := s.repo.FindByPhone(ctx, phone)
	isNew := false

	if err == domain.ErrNotFound {
		isNew = true
		p = &domain.Provider{
			Phone:     phone,
			Services:  []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return "", false, err
		}
	}

	token, err := s.token.GenerateProviderToken(p.ID)
	return token, isNew, err
}

func (s *ProviderService) GetProfile(ctx context.Context, id domain.ProviderID) (*domain.Provider, error) {
	return s.repo.FindByID(ctx, id)
}