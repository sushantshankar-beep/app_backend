package service

import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceService struct {
	repo *repository.InvoiceRepo
}

func NewInvoiceService(repo *repository.InvoiceRepo) *InvoiceService {
	return &InvoiceService{repo: repo}
}

func (s *InvoiceService) GenerateInvoice(
	ctx context.Context,
	userID string,
	serviceID string,
	items []domain.InvoiceItem,
) (*domain.Invoice, error) {

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var subTotal float64
	var tax float64
	for _, i := range items {
		subTotal += i.Price
		tax += (i.Price * i.GSTPercent) / 100
	}

	inv := &domain.Invoice{
		InvoiceNumber: fmt.Sprintf("INV-%d", time.Now().Unix()),
		UserID:        userOID,
		ServiceID:     serviceOID,
		Items:         items,
		SubTotal:      subTotal,
		TaxAmount:     tax,
		TotalAmount:   subTotal + tax,
		CreatedAt:     time.Now(),
	}

	go generateInvoicePDF(inv)

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}

	return inv, nil
}

func (s *InvoiceService) GetInvoice(ctx context.Context, serviceID string) (*domain.Invoice, error) {
	return s.repo.FindByServiceID(ctx, serviceID)
}