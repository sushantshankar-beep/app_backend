package ports

import (
	"context"

	"app_backend/internal/domain"
)

type ProviderTimeoutHandler interface {
	HandleProviderTimeout(ctx context.Context, svc *domain.AcceptedService)
}
