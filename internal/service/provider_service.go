package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/worker"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"app_backend/internal/repository"
	"errors"
	"strings"
	"fmt"
)

type ProviderService struct {
	repo                ports.ProviderRepository
	counterRepo *repository.CounterRepo
	otp                 ports.OTPStore
	token               ports.TokenService
	queue               *worker.OTPQueue
	acceptedServiceRepo ports.AcceptedServiceRepository
}

func NewProviderService(repo ports.ProviderRepository,counterRepo *repository.CounterRepo,otp ports.OTPStore,token ports.TokenService,q *worker.OTPQueue,acceptedRepo ports.AcceptedServiceRepository) *ProviderService {
	return &ProviderService{
		repo:                repo,
		counterRepo:		counterRepo,
		otp:                 otp,
		token:               token,
		queue:               q,
		acceptedServiceRepo: acceptedRepo,
	}
}
func assignString(dst *string, v any) {
	if s, ok := v.(string); ok && s != "" {
		*dst = s
	}
}
func toStringSlice(src []any) []string {
	out := make([]string, 0, len(src))
	for _, v := range src {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseProofs(src []any) []domain.Proof {
	out := make([]domain.Proof, 0, len(src))
	for _, v := range src {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, domain.Proof{
			Type:     m["type"].(string),
			File:     m["file"].(string),
			Verified: "pending",
		})
	}
	return out
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
) (string, bool, error) {

	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return "", false, domain.ErrOTPInvalid
	}
	if time.Now().After(otp.ExpiresAt) {
		return "", false, domain.ErrOTPExpired
	}
	provider, err := s.repo.FindByPhone(ctx, phone)

	isNew := false
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
			return "", false, err
		}
	}

	// ✅ GENERATE PROVIDER JWT
	token, err := s.token.GenerateProviderToken(provider.ID)
	return token, isNew, err
}


/* ---------------- PROFILE ---------------- */

func (s *ProviderService) GetProfile(ctx context.Context, id domain.ProviderID) (*domain.Provider, error) {
	return s.repo.FindByID(ctx, id)
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

	// ================= REQUIRED STRING FIELDS =================
	requiredStrings := []string{"name","email","companyName","profileUrl","address","permanentAddress"}

	for _, field := range requiredStrings {
		v, ok := req[field]
		if !ok {
			return nil, errors.New(field + " is required")
		}
		str, ok := v.(string)
		if !ok || strings.TrimSpace(str) == "" {
			return nil, errors.New(field + " must be a non-empty string")
		}
	}

	// ================= REQUIRED ARRAY FIELD =================
	vehicleTypeRaw, ok := req["vehicleType"].([]any)
	if !ok || len(vehicleTypeRaw) == 0 {
		return nil, errors.New("vehicleType is required and must have at least one value")
	}

	// ================= ASSIGN CORE =================
	assignString(&provider.Name, req["name"])
	assignString(&provider.Email, req["email"])
	assignString(&provider.CompanyName, req["companyName"])
	assignString(&provider.ProfileURL, req["profileUrl"]) // S3 URL
	assignString(&provider.Address, req["address"])
	assignString(&provider.PermanentAddress, req["permanentAddress"])
	assignString(&provider.AlternateContact, req["alternateContact"])
	assignString(&provider.City, req["city"])
	assignString(&provider.VehicleNumber, req["vehicleNumber"])
	assignString(&provider.Description, req["description"])

	// ================= ASSIGN ARRAYS =================
	provider.VehicleType = toStringSlice(vehicleTypeRaw)

	if v, ok := req["providerServices"].([]any); ok {
		provider.ProviderServices = toStringSlice(v)
	}
	if v, ok := req["providerBrands"].([]any); ok {
		provider.ProviderBrands = toStringSlice(v)
	}

	// ================= PROOFS (OPTIONAL) =================
	if v, ok := req["identityProof"].([]any); ok {
		provider.IdentityProof = parseProofs(v)
	}
	if v, ok := req["addressProof"].([]any); ok {
		provider.AddressProof = parseProofs(v)
	}

	// ================= FINAL STATE =================
	provider.FormSubmitted = 1
	provider.IsActive = "active"
	provider.UpdatedAt = time.Now()

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
