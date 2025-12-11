package ports

import (
    "app_backend/internal/domain"
    "context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) error
	FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.User, error)

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
    Count(ctx context.Context, filter bson.M) (int64, error)
    ListByProvider(ctx context.Context, providerID domain.ProviderID, skip, limit int) ([]domain.AcceptedService, error)
    FindByIDAndProvider(ctx context.Context, id string, providerID domain.ProviderID) (*domain.AcceptedService, error)
    Aggregate(ctx context.Context, pipeline mongo.Pipeline) ([]map[string]any, error)
}

type HomepageRepository interface {
	Create(ctx context.Context, h *domain.Homepage) error
	Update(ctx context.Context, h *domain.Homepage) error
	FindByID(ctx context.Context, id string) (*domain.Homepage, error)
	FindAll(ctx context.Context, skip, limit int) ([]domain.Homepage, error)
	Count(ctx context.Context, filter bson.M) (int64, error)
}
type ServiceRequestRepository interface {
	Create(ctx context.Context, s *domain.ServiceRequest) error
	Update(ctx context.Context, s *domain.ServiceRequest) error
	FindByID(ctx context.Context, id string) (*domain.ServiceRequest, error)
	FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.ServiceRequest, error)

	
}

type SavedVehicleRepository interface {
	FindByUserAndVehicleNumber(ctx context.Context, userID string, vehicleNumber string) (*domain.SavedVehicleData, error)
	Create(ctx context.Context, v *domain.SavedVehicleData) error
}

type BidRepository interface {
	FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.Bid, error)
}


