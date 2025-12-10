package service

import (
	"context"

	"app_backend/internal/domain"
)

type UserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) error
}

type ProviderRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.Provider, error)
	FindByID(ctx context.Context, id domain.ProviderID) (*domain.Provider, error)
	Create(ctx context.Context, p *domain.Provider) error
	Update(ctx context.Context, p *domain.Provider) error
}

type OTPStore interface {
	Save(ctx context.Context, otp *domain.OTP) error
	Find(ctx context.Context, phone, code string) (*domain.OTP, error)
	Delete(ctx context.Context, phone string) error
}

type SMSClient interface {
	SendOTP(ctx context.Context, phone, msg string) error
}

type TokenService interface {
	GenerateUserToken(id domain.UserID) (string, error)
	GenerateProviderToken(id domain.ProviderID) (string, error)
	Parse(tokenString string) (id string, typ string, err error)
}
