package ports

import "context"

type BiddingService interface {
	FindMechanics(ctx context.Context, requestID string, lat, lng float64) error
	PlaceBid(ctx context.Context, requestID, providerID string, amount float64) error
	AcceptBid(ctx context.Context, requestID, providerID string) error
}
