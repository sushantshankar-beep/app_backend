package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"app_backend/internal/domain"
)

type RefundProcessor interface {
	ProcessRefund(
		ctx context.Context,
		mihpayid string,
		amount float64,
	) (*domain.PayURefundResponse, error)
}

type RefundRepo interface {
	UpdateByMihPayID(
		ctx context.Context,
		mihpayid string,
		update bson.M,
	) error
}
