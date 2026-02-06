package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/dto"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/s3"
	"app_backend/internal/utils"
	"app_backend/internal/worker"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderService struct {
	repo                ports.ProviderRepository
	counterRepo         *repository.CounterRepo
	otp                 ports.OTPStore
	token               ports.TokenService
	queue               *worker.OTPQueue
	acceptedServiceRepo ports.AcceptedServiceRepository
	providerRepo        *repository.ProviderRepo
	kycRepo             *repository.KYCRepo
	userRepo            *repository.UserRepo
	ratingsRepo         *repository.RatingRepo
	agreementRepo       *repository.AgreementRepo
	pdfUploader         *s3.PDFUploader
}

func NewProviderService(repo ports.ProviderRepository, counterRepo *repository.CounterRepo, otp ports.OTPStore, token ports.TokenService, q *worker.OTPQueue, acceptedRepo ports.AcceptedServiceRepository, providerRepo *repository.ProviderRepo, kycRepo *repository.KYCRepo, userRepo *repository.UserRepo, ratingsRepo *repository.RatingRepo, agreementRepo *repository.AgreementRepo, pdfUploader *s3.PDFUploader) *ProviderService {
	return &ProviderService{
		repo:                repo,
		counterRepo:         counterRepo,
		otp:                 otp,
		token:               token,
		queue:               q,
		acceptedServiceRepo: acceptedRepo,
		providerRepo:        providerRepo,
		kycRepo:             kycRepo,
		userRepo:            userRepo,
		ratingsRepo:         ratingsRepo,
		agreementRepo:       agreementRepo,
		pdfUploader:         pdfUploader,
	}
}
func isProviderProfileCompleted(p *domain.Provider) bool {
	if p.Name == "" {
		return false
	}
	if p.City == "" {
		return false
	}
	return true
}


func assignString(field *string, val any) {
	if val == nil {
		return
	}
	str, ok := val.(string)
	if !ok {
		return
	}
	if strings.TrimSpace(str) != "" {
		*field = str
	}
}


func toStringSlice(slice []any) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

/* ---------------- OTP ---------------- */

func (s *ProviderService) SendOTP(ctx context.Context, phone string) error {
	code := GenerateOTP()

	otp := &domain.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.otp.Save(ctx, otp); err != nil {
		return err
	}

	s.queue.Enqueue(worker.OTPJob{
		Phone: phone,
		Msg:   code,
		Type:  "provider",
	})

	return nil
}

func (s *ProviderService) VerifyOTP(
	ctx context.Context,
	phone, code string,
) (string, bool,bool, error) {

	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return "", false, false, domain.ErrOTPInvalid
	}
	if time.Now().After(otp.ExpiresAt) {
		return "", false,false,domain.ErrOTPExpired
	}
	provider, err := s.repo.FindByPhone(ctx, phone)
	var(
		isNew  bool
		isProfileCompleted bool
	)
	if err == domain.ErrNotFound {
		isNew = true

		seq, _ := s.counterRepo.Next(ctx, "provider")
		providerCode := fmt.Sprintf("PRV-%05d", seq)

		provider = &domain.Provider{
			Phone:      phone,
			ProviderCode: providerCode,
			IsActive:   "inactive",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.repo.Create(ctx, provider); err != nil {
			return "", false,false,err
		}
		isProfileCompleted = false
	}else if err != nil{
		return "",false,false,err
	}else{
		if provider.IsActive == domain.PROVIDER_DELETED {
			isNew = true
			isProfileCompleted = false
		} else {
			isNew = false
			isProfileCompleted = isProviderProfileCompleted(provider)
		}
	}

	// ✅ GENERATE PROVIDER JWT
	token, err := s.token.GenerateProviderToken(provider.ID)
	return token, isNew,isProfileCompleted,err
}



/* ---------------- PROFILE ---------------- */

func (s *ProviderService) GetProfile(
	ctx context.Context,
	id domain.ProviderID,
) (*domain.Provider, error) {

	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	provider.KycStatus = domain.KYC_PENDING

	if provider.KYCID != primitive.NilObjectID {
		kyc, err := s.kycRepo.FindByProviderID(
			ctx,
			string(provider.ID),
		)

		if err == nil && kyc != nil {
			provider.KycStatus = kyc.Status
		}
	}

	return provider, nil
}

func (s *ProviderService) CreateOrUpdateProfile(
	ctx context.Context,
	id domain.ProviderID,
	req map[string]any,
) (*domain.Provider, error) {

	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	isDeleted := provider.IsActive == domain.PROVIDER_DELETED
	wasCompleted := isProviderProfileCompleted(provider)

	if !wasCompleted && !isDeleted {
		required := []string{"name", "city", "shopAddress"}
		for _, field := range required {
			v, ok := req[field]
			if !ok {
				return nil, errors.New(field + " is required")
			}
			str, ok := v.(string)
			if !ok || strings.TrimSpace(str) == "" {
				return nil, errors.New(field + " must be a non-empty string")
			}
		}
	}

	keyFieldsChanged := false
	oldName := provider.Name
	oldCompanyName := provider.CompanyName
	oldCity := provider.City

	assignString(&provider.Name, req["name"])
	assignString(&provider.Email, req["email"])
	assignString(&provider.CompanyName, req["companyName"])
	assignString(&provider.ProfileURL, req["profileUrl"])
	assignString(&provider.Address, req["address"])
	assignString(&provider.PermanentAddress, req["shopAddress"])
	assignString(&provider.AlternateContact, req["alternateContact"])
	assignString(&provider.City, req["city"])
	assignString(&provider.VehicleNumber, req["vehicleNumber"])
	assignString(&provider.Description, req["description"])

	if provider.Name != oldName || 
	   provider.CompanyName != oldCompanyName || 
	   provider.City != oldCity {
		keyFieldsChanged = true
	}

	if v, ok := req["vehicleType"]; ok {
		switch t := v.(type) {
		case []any:
			provider.VehicleType = toStringSlice(t)
		case []string:
			provider.VehicleType = t
		}
	}

	if v, ok := req["providerServices"]; ok {
		switch t := v.(type) {
		case []any:
			provider.ProviderServices = toStringSlice(t)
		case []string:
			provider.ProviderServices = t
		}
	}

	if v, ok := req["providerBrands"]; ok {
		switch t := v.(type) {
		case []any:
			provider.ProviderBrands = toStringSlice(t)
		case []string:
			provider.ProviderBrands = t
		}
	}

	provider.UpdatedAt = time.Now()

	shouldGenerateAgreement := false
	
	if isDeleted {
		provider.IsActive = domain.PROVIDER_ACTIVE
		provider.FormSubmitted = 1
		shouldGenerateAgreement = true
	} else if !wasCompleted && isProviderProfileCompleted(provider) {
		provider.FormSubmitted = 1
		provider.IsActive = domain.PROVIDER_ACTIVE
		shouldGenerateAgreement = true
	} else if keyFieldsChanged && isProviderProfileCompleted(provider) {
		shouldGenerateAgreement = true
	}

	if shouldGenerateAgreement {
		if err := s.generateAndUploadAgreement(ctx, provider); err != nil {
			return nil, fmt.Errorf("failed to generate agreement: %w", err)
		}
	}

	if err := s.repo.Update(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

/* ---------------- SERVICES ---------------- */

func (s *ProviderService) GetMyAllServices(
	ctx context.Context,
	providerID domain.ProviderID,
	page, limit int,
) (map[string][]map[string]any, int64, error) {

	skip := (page - 1) * limit

	services, err := s.acceptedServiceRepo.ListByProvider(ctx, providerID, skip, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.acceptedServiceRepo.Count(ctx, bson.M{
		"provider":      providerID,
		"paymentStatus": "paid",
	})
	if err != nil {
		return nil, 0, err
	}

	grouped := map[string][]map[string]any{
		"ongoing":   {},
		"completed": {},
		"cancelled": {},
	}

	for _, svc := range services {
		item := map[string]any{
			"id":         svc.ID,
			"finalPrice": svc.FinalPrice,
			"basePrice":  svc.BasePrice,
			"issues":     svc.Issues,
			"createdAt":  svc.CreatedAt,
			"status":     svc.Status,
		}

		switch {
		case ongoingSet[svc.Status]:
			grouped["ongoing"] = append(grouped["ongoing"], item)
		case completedSet[svc.Status]:
			grouped["completed"] = append(grouped["completed"], item)
		default:
			grouped["cancelled"] = append(grouped["cancelled"], item)
		}
	}

	return grouped, total, nil
}

func (s *ProviderService) GetMyService(
	ctx context.Context,
	providerID domain.ProviderID,
	serviceID string,
) (*domain.AcceptedService, error) {
	return s.acceptedServiceRepo.FindByIDAndProvider(ctx, serviceID, providerID)
}

/* ---------------- STATE MAPS ---------------- */

var (
	ongoingSet = map[domain.ServiceStatus]bool{
		domain.StatusNotStarted:      true,
		domain.StatusStarted:         true,
		domain.StatusReachedLocation: true,
		domain.StatusOTPVerified:     true,
		domain.StatusInProgress:      true,
	}

	completedSet = map[domain.ServiceStatus]bool{
		domain.StatusCompleted: true,
	}
)



/* ---------------- ACTIONS ---------------- */

func (s *ProviderService) StartService(ctx context.Context,serviceID string,providerID domain.ProviderID) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	if err := domain.CanStartService(svc); err != nil {
		return err
	}

	return s.acceptedServiceRepo.UpdateByID(ctx, serviceOID, bson.M{
		"$set": bson.M{
			"status":    domain.StatusInProgress,
			"startedAt": time.Now(),
		},
	})
}

func (s *ProviderService) CompleteService(ctx context.Context,serviceID string) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	if err := domain.CanCompleteService(svc); err != nil {
		return err
	}

	return s.acceptedServiceRepo.UpdateByID(ctx, serviceOID, bson.M{
		"$set": bson.M{
			"status":      domain.StatusCompleted,
			"completedAt": time.Now(),
		},
	})
}


func (s *ProviderService) SubmitAgreement(ctx context.Context,id domain.ProviderID ) (*domain.Provider, error) {

	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if provider.IsAgreementSubmitted {
		return nil, domain.ErrAlreadySubmitted
	}

	now := time.Now()

	provider.IsAgreementSubmitted = true
	provider.AgreementSubmittedAt = &now
	provider.UpdatedAt = now

	if err := s.generateAndUploadAgreement(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to generate agreement with submission date: %w", err)
	}

	if err := s.providerRepo.UpdateAgreement(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *ProviderService) Logout(ctx context.Context, providerID domain.ProviderID, token string) error {
	return nil
}

func (s *ProviderService) DeleteProviderAccount(ctx context.Context, id domain.ProviderID) error {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	provider.IsActive = domain.PROVIDER_DELETED
	provider.FormSubmitted = 0
	provider.UpdatedAt = time.Now()

	return s.repo.Update(ctx, provider)
}

func (s *ProviderService) GetProviderServicesAndReviews(ctx context.Context, providerID string) (*dto.ProviderServicesAndReviewsResponse, error) {
	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, errors.New("invalid provider ID")
	}

	provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(providerID))
	if err != nil {
		return nil, errors.New("provider not found")
	}

	ratings, err := s.ratingsRepo.GetRatingsByRatee(ctx, providerObjID, domain.RATING_PROVIDER)
	if err != nil {
		return nil, err
	}

	uniqueRatings := s.getUniqueRatings(ratings)

	var total int
	for _, r := range uniqueRatings {
		total += int(r.Stars)
	}

	avg := 0.0
	if len(uniqueRatings) > 0 {
		avg = float64(total) / float64(len(uniqueRatings))
	}

	topRatings := uniqueRatings
	if len(uniqueRatings) > 5 {
		topRatings = uniqueRatings[:5]
	}

	topReviews := make([]dto.RatingResponse, 0, len(topRatings))
	for _, r := range topRatings {
		var raterName, raterProfileImage string
		if user, err := s.userRepo.GetByID(ctx, r.RaterID); err == nil && user != nil {
			raterName = user.Name
			raterProfileImage = user.ImageUrl
		}

		topReviews = append(topReviews, dto.RatingResponse{
			ID:                r.ID.Hex(),
			RaterID:           r.RaterID.Hex(),
			RaterName:         raterName,
			RaterProfileImage: raterProfileImage,
			RateeID:           r.RateeID.Hex(),
			RateeName:         provider.Name,
			RatingType:        string(r.RatingType),
			Stars:             r.Stars,
			Review:            r.Review,
			Recommended:       *r.Recommended,
			CreatedAt:         r.CreatedAt,
		})
	}

	return &dto.ProviderServicesAndReviewsResponse{
		Services:      provider.ProviderServices,
		AverageRating: avg,
		TotalRatings:  len(uniqueRatings),
		TopReviews:    topReviews,
	}, nil
}

func (s *ProviderService) getUniqueRatings(ratings []domain.Rating) []domain.Rating {
	seen := make(map[string]bool)
	unique := make([]domain.Rating, 0)

	for _, r := range ratings {
		raterID := r.RaterID.Hex()
		if !seen[raterID] {
			seen[raterID] = true
			unique = append(unique, r)
		}
	}

	return unique
}

func (s *ProviderService) GetAgreementTemplate(ctx context.Context) (*domain.Agreement, error) {
	return s.agreementRepo.FindDefault(ctx)
}

func (s *ProviderService) generateAndUploadAgreement(ctx context.Context, provider *domain.Provider) error {
	agreement, err := s.agreementRepo.FindDefault(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch agreement template: %w", err)
	}

	pdfBytes, err := utils.GenerateAgreementPDF(provider, agreement)
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	url, err := s.pdfUploader.UploadPDF(pdfBytes, string(provider.ID))
	if err != nil {
		return fmt.Errorf("failed to upload PDF: %w", err)
	}

	provider.AgreementUrl = url
	return nil
}
