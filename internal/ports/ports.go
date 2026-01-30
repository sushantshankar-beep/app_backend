
package ports
import (
    "app_backend/internal/domain"
    "context"
	"go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/bson"
    "time"
)

type UserRepository interface {
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, update bson.M) (*domain.User, error)
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
	SendOTP(ctx context.Context, phone, msg string,Type string) error
}

type TokenService interface {
	GenerateUserToken(id domain.UserID) (string, error)
	GenerateProviderToken(id domain.ProviderID) (string, error)
	Parse(token string) (string, string, error)
}

type AcceptedServiceRepository interface {
	Create(ctx context.Context, svc *domain.AcceptedService) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.AcceptedService, error)
	Find(ctx context.Context, filter bson.M, skip, limit int) ([]domain.AcceptedService, error)
	Count(ctx context.Context, filter bson.M) (int64, error)
	ListByProvider(ctx context.Context, providerID domain.ProviderID, skip, limit int) ([]domain.AcceptedService, error)
	FindByIDAndProvider(ctx context.Context, serviceID string, providerID domain.ProviderID) (*domain.AcceptedService, error)

	UpdateByID(ctx context.Context, id primitive.ObjectID, update bson.M) error
	UpdatePaymentStatus(ctx context.Context, id primitive.ObjectID, status domain.PaymentStatus) error
	FindStaleSearching(ctx context.Context, before time.Time) ([]domain.AcceptedService, error)
	FindStuckAssigned(
		ctx context.Context,
		before time.Time,
	) ([]*domain.AcceptedService, error)
	UpdateStatus(
		ctx context.Context,
		id primitive.ObjectID,
		status domain.ServiceStatus,
		fields bson.M,
	) error
}

type HomepageRepository interface {
	Create(ctx context.Context, h *domain.Homepage) error
	Update(ctx context.Context, h *domain.Homepage) error
	FindByID(ctx context.Context, id string) (*domain.Homepage, error)
	FindAll(ctx context.Context, skip, limit int) ([]domain.Homepage, error)
	Count(ctx context.Context, filter bson.M) (int64, error)
}