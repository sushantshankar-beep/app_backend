package ports

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

type AcceptedServiceRepo interface {
	Col() *mongo.Collection

	// Existing domain-level queries (used elsewhere)
	FindByIDAndProvider(
		ctx context.Context,
		id string,
		providerID domain.ProviderID,
	) (*domain.AcceptedService, error)
}
