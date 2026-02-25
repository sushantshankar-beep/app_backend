package service

import (
	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/s3"
	"app_backend/internal/utils"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceService struct {
	repo            *repository.InvoiceRepo
	serviceRepo     *repository.AcceptedServiceRepo
	userRepo        *repository.UserRepo
	providerRepo    *repository.ProviderRepo
	transactionRepo *repository.PaymentRepository

	s3Uploader *s3.InvoiceUploader
	bucket     string
	s3Folder   string
}

func NewInvoiceService(
	repo *repository.InvoiceRepo,
	serviceRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	transactionRepo *repository.PaymentRepository,
	s3Uploader *s3.InvoiceUploader,
) *InvoiceService {

	return &InvoiceService{
		repo:            repo,
		serviceRepo:     serviceRepo,
		userRepo:        userRepo,
		providerRepo:    providerRepo,
		transactionRepo: transactionRepo,
		s3Uploader:      s3Uploader,
	}
}

func (s *InvoiceService) GenerateInvoice(
	ctx context.Context,
	userID string,
	serviceID string,
) (*domain.Invoice, error) {

	return s.GenerateInternal(ctx, userID, serviceID)
}


func (s *InvoiceService) GenerateInternal(ctx context.Context,userID string, serviceID string ) (*domain.Invoice, error) {
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

	const gstPercent = 18.0

	serviceCharge := service.FinalPrice
	totalDiscount := service.TotalDiscount

	hasDiscount :=
		totalDiscount > 0 ||
			service.AppliedPromo != nil ||
			service.AppliedDiscount != nil

	var amountAfterDiscount float64
	var gst float64
	var total float64

	if hasDiscount {
		gst = (serviceCharge * gstPercent) / 100
		amountAfterDiscount = serviceCharge - totalDiscount
		total = serviceCharge + gst - totalDiscount
	} else {
		gst = (serviceCharge * gstPercent) / 100
		total = serviceCharge + gst
	}

	var appliedPromo *domain.AppliedPromoSummary
	if service.AppliedPromo != nil {
		appliedPromo = &domain.AppliedPromoSummary{
			Code:        service.AppliedPromo.Code,
			DiscountAmt: service.AppliedPromo.DiscountAmt,
		}
	}

	var appliedDiscount *domain.AppliedDiscountSummary
	if service.AppliedDiscount != nil {
		appliedDiscount = &domain.AppliedDiscountSummary{
			Code:        service.AppliedDiscount.Code,
			DiscountAmt: service.AppliedDiscount.DiscountAmt,
		}
	}

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
			ServiceCharge:       utils.RoundTo2(serviceCharge),
			TotalDiscount:       utils.RoundTo2(totalDiscount),
			AmountAfterDiscount: utils.RoundTo2(amountAfterDiscount),
			GSTPercent:          gstPercent,
			GST:                 utils.RoundTo2(gst),
			Total:               utils.RoundTo2(total),
			AppliedPromo:        appliedPromo,
			AppliedDiscount:     appliedDiscount,
		},
		Transaction:domain.InvoiceTransaction{
			PaymentMode: transaction.Method,
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
	defer os.Remove(htmlPath)

	pdfURL, err := s.convertHTMLToPDFAndUpload(htmlPath)
	if err != nil {
		return nil, err
	}

	inv.PDFUrl = pdfURL

	_ = s.repo.UpdatePDFUrl(ctx, inv.ID, pdfURL)

	return inv, nil
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

func (s *InvoiceService) GetInvoicePDF(
	ctx context.Context,
	bookingId string,
) (string, error) {

	invoice, err := s.GetInvoice(ctx, bookingId)
	if err != nil {
		return "", err
	}

	if invoice.PDFUrl == "" {
		return "", fmt.Errorf("invoice PDF not generated")
	}

	return invoice.PDFUrl, nil
}
