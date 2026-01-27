package ports

import "context"

type SearchingServiceCleaner interface {
	CancelSearchingService(ctx context.Context, serviceID string, userID string) error
}
