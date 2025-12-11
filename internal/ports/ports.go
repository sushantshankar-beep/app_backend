package ports

import (
    "app_backend/internal/domain"
    "context"
    "go.mongodb.org/mongo-driver/bson"
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
	Parse(token string) (string, string, error)
}

type AcceptedServiceRepository interface {
    Find(ctx context.Context, filter bson.M, skip, limit int) ([]domain.AcceptedService, error)
    ListByProvider(ctx context.Context, providerID domain.ProviderID, skip, limit int) ([]domain.AcceptedService, error)
    FindByIDAndProvider(ctx context.Context, id string, providerID domain.ProviderID) (*domain.AcceptedService, error)
    Count(ctx context.Context, filter bson.M) (int64, error)
}
type HomepageRepository interface {
	Create(ctx context.Context, h *domain.Homepage) error
	Update(ctx context.Context, h *domain.Homepage) error
	FindByID(ctx context.Context, id string) (*domain.Homepage, error)
	FindAll(ctx context.Context, skip, limit int) ([]domain.Homepage, error)
	Count(ctx context.Context, filter bson.M) (int64, error)
}