package service

import (
	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/utils"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceService struct {
	repo         *repository.InvoiceRepo
	serviceRepo  *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	transactionRepo *repository.PaymentRepository
}

func NewInvoiceService(
	repo *repository.InvoiceRepo,
	serviceRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	transactionRepo *repository.PaymentRepository,
) *InvoiceService {
	return &InvoiceService{
		repo:         repo,
		serviceRepo:  serviceRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		transactionRepo: transactionRepo,
	}
}

func (s *InvoiceService) GenerateInvoice(ctx context.Context,userID string, serviceID string ) (*domain.Invoice, error) {
	userOID, err := primitive.ObjectIDFromHex(userID)
	defer func() {
		if r := recover(); r != nil {
			log.Println("🔥 panic in GenerateInvoice:", r)
		}
	}()
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
	transaction , err := s.transactionRepo.GetLatestPaidTransactionByServiceID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("invoice cannot be generated without successful payment: %w", err)
	}

	finalPrice := service.FinalPrice
	gst := finalPrice * 0.18

	inv := &domain.Invoice{
		InvoiceNumber: s.generateInvoiceNumber(service.ID),
		InvoiceDate:   time.Now(),
		ServiceDate:   service.CompletedAt,
		UserID:        userOID,
		ServiceID:     service.ID.Hex(),
        ServiceNumber: service.ServiceNumber,
		CompanyInfo: domain.CompanyInfo{
			Name:    "Vahanwire",
			Address: "B819 Noida One Tower B Noida Sector 62 Uttar Pradesh, 201301",
			GST:     "09AAGCI0467A1ZV",
			Phone:   "0120 3221368",
			Email:   "Info@vahanwire.com",
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
			ServiceCharge: utils.RoundTo2(finalPrice),
			GST:           utils.RoundTo2(gst),
			Total:         utils.RoundTo2(finalPrice + gst),
		},
		Transaction:domain.Transaction{
			PaymentMode: transaction.PaymentMode,
		},
		CreatedAt: time.Now(),
	}
    
	
	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to save invoice: %w", err)
	}

    htmlPath, err := RenderInvoiceHTML(inv)
    if err != nil {
	  return nil, err
    }
  
    fileName, err := ConvertHTMLToPDF(htmlPath)
    if err != nil {
	   return nil, err
    }

    defer os.Remove(htmlPath)

    baseURL := os.Getenv("INVOICE_BASE_URL")
   
    publicURL := fmt.Sprintf("%s/invoices/%s", baseURL, fileName)
    inv.PDFUrl = publicURL 

    _ = s.repo.UpdatePDFUrl(ctx, inv.ID, publicURL)
    return  inv, nil
}

func (s *InvoiceService) generateInvoiceNumber(serviceID primitive.ObjectID) string {
	hexID := serviceID.Hex()
	if len(hexID) >= 8 {
		return fmt.Sprintf("INV-%s", hexID[len(hexID)-8:])
	}
	return fmt.Sprintf("INV-%s", hexID)
}

func (s *InvoiceService) GetInvoice(ctx context.Context, bookingID string) (*domain.Invoice, error) {
	return s.repo.GetByID(ctx, bookingID)
}

func (s *InvoiceService) GetInvoicePDF(ctx context.Context, bookingId string) (string, string, error) {
	invoice, err := s.GetInvoice(ctx, bookingId)
	if err != nil {
		return "", "", err
	}

	if invoice.PDFUrl == "" {
		return "", "", fmt.Errorf("PDF not generated for invoice %s", bookingId)
	}

	filename := filepath.Base(invoice.PDFUrl)

	pdfPath := filepath.Join("internal/storage/invoices", filename)

	if _, err := os.Stat(pdfPath); err != nil {
		return "", "", fmt.Errorf("PDF file not found on server")
	}

	return pdfPath, filename, nil
}
