package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceService struct {
	repo         *repository.InvoiceRepo
	serviceRepo  *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	pdfService   *PDFService
}

func NewInvoiceService(
	repo *repository.InvoiceRepo,
	serviceRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
) *InvoiceService {

	pdfDir := "./invoices"
	os.MkdirAll(pdfDir, os.ModePerm)

	pdfService := NewPDFService(pdfDir)

	return &InvoiceService{
		repo:         repo,
		serviceRepo:  serviceRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		pdfService:   pdfService,
	}
}

func (s *InvoiceService) GenerateInvoice(
	ctx context.Context,
	userID string,
	serviceID string,
) (*domain.Invoice, error) {

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service ID: %w", err)
	}

	service, err := s.serviceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch service: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, service.User)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(service.Provider.Hex()))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider: %w", err)
	}

	finalPrice := service.FinalPrice
	gst := finalPrice * 0.18
	subTotal := finalPrice - gst

	inv := &domain.Invoice{
		InvoiceNumber: s.generateInvoiceNumber(service.ID),
		InvoiceDate:   time.Now(),
		ServiceDate:   service.CompletedAt,
		UserID:        userOID,
		ServiceID:     serviceOID,
		ProviderID:    string(provider.ID),
		ProviderInfo: domain.ProviderInfo{
			Name:    provider.CompanyName,
			Address: provider.Address,
			Phone:   provider.Phone,
			Email: provider.Email,
		},
		CustomerInfo: domain.CustomerInfo{
			Name:  user.Name,
			Phone: user.Phone,
			Email: user.Email,
		},
		VehicleDetails: domain.VehicleInfo{
			Brand:         service.Brand,
			Model:         service.Model,
			Year:          service.ModelYear,
			VehicleType:   service.VehicleType,
			VehicleNumber: service.VehicleNumber,
			FuelType:      service.FuelType,
		},
		ServiceInfo: domain.ServiceInfo{
			Type:          service.ServiceType,
			Status:        service.Status,
			PaymentStatus: service.PaymentStatus,
		},
		PricingDeatils: domain.PricingInfo{
			ServiceCharge: finalPrice,
			Discount:      0,
			SubTotal:      subTotal,
			GST:           gst,
			Total:         finalPrice,
		},
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

	pdfPath, err := s.pdfService.GenerateInvoicePDF(inv)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	inv.PDFUrl = pdfPath
	if err := s.repo.UpdatePDFUrl(ctx, inv.ID, pdfPath); err != nil {
		return nil, fmt.Errorf("failed to update PDF URL: %w", err)
	}

	return inv, nil
}

func (s *InvoiceService) generateInvoiceNumber(serviceID primitive.ObjectID) string {
	hexID := serviceID.Hex()
	if len(hexID) >= 8 {
		return fmt.Sprintf("INV-%s", hexID[len(hexID)-8:])
	}
	return fmt.Sprintf("INV-%s", hexID)
}

func (s *InvoiceService) GetInvoice(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	oid, err := primitive.ObjectIDFromHex(invoiceID)
	if err != nil {
		return nil, fmt.Errorf("invalid invoice ID: %w", err)
	}
	return s.repo.GetByID(ctx, oid)
}

func (s *InvoiceService) GetInvoicePDF(ctx context.Context, invoiceID string) (string, string, error) {
	invoice, err := s.GetInvoice(ctx, invoiceID)
	if err != nil {
		return "", "", err
	}

	if invoice.PDFUrl == "" {
		return "", "", fmt.Errorf("PDF not generated for invoice %s", invoiceID)
	}

	filename := filepath.Base(invoice.PDFUrl)
	if filename == "" {
		filename = fmt.Sprintf("%s.pdf", invoice.InvoiceNumber)
	}

	return invoice.PDFUrl, filename, nil
}
