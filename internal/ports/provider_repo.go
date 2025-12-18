package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type ProviderRepo interface {
	// Infra-level read (for FCM, assignment, etc.)
	FindOne(
		ctx context.Context,
		filter bson.M,
		result any,
	) error
}
