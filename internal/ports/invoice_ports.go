package ports

import (
	"context"
	"app_backend/internal/domain"
)

type InvoiceGenerator interface {
	GenerateInvoice(ctx context.Context, userID, serviceID string) (*domain.Invoice, error)
}
