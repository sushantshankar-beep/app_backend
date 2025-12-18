package ports

import "context"

type RefundProcessor interface {
	ProcessRefund(ctx context.Context, mihpayid string, amount float64) error
}
