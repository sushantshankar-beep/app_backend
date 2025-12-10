package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/worker"
	"go.mongodb.org/mongo-driver/bson"

)

type ProviderService struct {
	repo                ports.ProviderRepository
	otp                 ports.OTPStore
	token               ports.TokenService
	queue               *worker.OTPQueue
	AcceptedServiceRepo ports.AcceptedServiceRepository
}

func NewProviderService(
	repo ports.ProviderRepository,
	otp ports.OTPStore,
	token ports.TokenService,
	q *worker.OTPQueue,
	acceptedRepo ports.AcceptedServiceRepository,
) *ProviderService {
	return &ProviderService{
		repo:                repo,
		otp:                 otp,
		token:               token,
		queue:               q,
		AcceptedServiceRepo: acceptedRepo,
	}
}

func (s *ProviderService) SendOTP(ctx context.Context, phone string) error {
	code := "1234"

	otp := &domain.OTP{
		Phone:     phone,
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.otp.Save(ctx, otp); err != nil {
		return err
	}

	s.queue.Enqueue(worker.OTPJob{Phone: phone, Msg: "Your provider OTP is " + code})
	return nil
}

func (s *ProviderService) VerifyOTP(ctx context.Context, phone, code string) (string, bool, error) {
	otp, err := s.otp.Find(ctx, phone, code)
	if err != nil {
		return "", false, domain.ErrOTPInvalid
	}
	if time.Now().After(otp.ExpiresAt) {
		return "", false, domain.ErrOTPExpired
	}

	_ = s.otp.Delete(ctx, phone)

	p, err := s.repo.FindByPhone(ctx, phone)
	isNew := false

	if err == domain.ErrNotFound {
		isNew = true
		p = &domain.Provider{
			Phone:     phone,
			Services:  []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return "", false, err
		}
	}

	token, err := s.token.GenerateProviderToken(p.ID)
	return token, isNew, err
}

func (s *ProviderService) GetProfile(ctx context.Context, id domain.ProviderID) (*domain.Provider, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProviderService) CreateOrUpdateProfile(ctx context.Context, id domain.ProviderID, req map[string]any) (*domain.Provider, error) {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name, ok := req["name"].(string); ok && name != "" {
		provider.Name = name
	}
	if email, ok := req["email"].(string); ok && email != "" {
		provider.Email = email
	}
	if alternateContact, ok := req["alternateContact"].(string); ok && alternateContact != "" {
		provider.AlternateContact = alternateContact
	}
	if profileUrl, ok := req["profileUrl"].(string); ok && profileUrl != "" {
		provider.ProfileURL = profileUrl
	}
	if address, ok := req["address"].(string); ok && address != "" {
		provider.Address = address
	}
	if permanentAddress, ok := req["permanentAddress"].(string); ok && permanentAddress != "" {
		provider.PermanentAddress = permanentAddress
	}
	if city, ok := req["city"].(string); ok && city != "" {
		provider.City = city
	}
	if gstNumber, ok := req["gstNumber"].(string); ok && gstNumber != "" {
		provider.GSTNumber = gstNumber
	}
	if vehicleNumber, ok := req["vehicleNumber"].(string); ok && vehicleNumber != "" {
		provider.VehicleNumber = vehicleNumber
	}
	if description, ok := req["description"].(string); ok && description != "" {
		provider.Description = description
	}
	if companyName, ok := req["companyName"].(string); ok && companyName != "" {
		provider.CompanyName = companyName
	}

	if vehicleType, ok := req["vehicleType"].([]any); ok && len(vehicleType) > 0 {
		provider.VehicleType = make([]string, len(vehicleType))
		for i, vt := range vehicleType {
			provider.VehicleType[i] = vt.(string)
		}
	}

	if providerServices, ok := req["providerServices"].([]any); ok && len(providerServices) > 0 {
		provider.ProviderServices = make([]string, len(providerServices))
		for i, ps := range providerServices {
			provider.ProviderServices[i] = ps.(string)
		}
	}
	if providerBrands, ok := req["providerBrands"].([]any); ok && len(providerBrands) > 0 {
		provider.ProviderBrands = make([]string, len(providerBrands))
		for i, pb := range providerBrands {
			provider.ProviderBrands[i] = pb.(string)
		}
	}

	if identityProof, ok := req["identityProof"].([]any); ok && len(identityProof) > 0 {
		provider.IdentityProof = make([]domain.Proof, len(identityProof))
		for i, ip := range identityProof {
			proofMap := ip.(map[string]any)
			provider.IdentityProof[i] = domain.Proof{
				Type:     proofMap["type"].(string),
				File:     proofMap["file"].(string),
				Verified: "pending",
			}
		}
	}

	if addressProof, ok := req["addressProof"].([]any); ok && len(addressProof) > 0 {
		provider.AddressProof = make([]domain.Proof, len(addressProof))
		for i, ap := range addressProof {
			proofMap := ap.(map[string]any)
			provider.AddressProof[i] = domain.Proof{
				Type:     proofMap["type"].(string),
				File:     proofMap["file"].(string),
				Verified: "pending",
			}
		}
	}

	if cancelCheque, ok := req["cancelCheque"].(map[string]any); ok && len(cancelCheque) > 0 {
		if file, ok := cancelCheque["file"].(string); ok && file != "" {
			provider.CancelCheque = domain.CancelCheque{
				File:     file,
				Verified: "pending",
			}
		}
	}

	if bankDetails, ok := req["bankDetails"].(map[string]any); ok && len(bankDetails) > 0 {
		if accountHolderName, ok := bankDetails["accountHolderName"].(string); ok && accountHolderName != "" {
			provider.BankDetails.AccountHolderName = accountHolderName
		}
		if accountNumber, ok := bankDetails["accountNumber"].(string); ok && accountNumber != "" {
			provider.BankDetails.AccountNumber = accountNumber
		}
		if ifscCode, ok := bankDetails["ifscCode"].(string); ok && ifscCode != "" {
			provider.BankDetails.IFSCCode = ifscCode
		}
		if branchName, ok := bankDetails["branchName"].(string); ok && branchName != "" {
			provider.BankDetails.BranchName = branchName
		}
		if upi, ok := bankDetails["upi"].(string); ok && upi != "" {
			provider.BankDetails.UPI = upi
		}
	}

	provider.FormSubmitted++
	provider.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *ProviderService) GetMyAllServices(ctx context.Context, providerID domain.ProviderID, page, limit int) (map[string][]map[string]any, int64, error) {
	skip := (page - 1) * limit

	services, err := s.AcceptedServiceRepo.ListByProvider(ctx, providerID, skip, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.AcceptedServiceRepo.Count(ctx, bson.M{"provider": providerID})
	if err != nil {
		return nil, 0, err
	}

	ongoingStatuses := []string{"not_started", "started", "reached_location", "otp_verified", "in_progress"}
	completedStatuses := []string{"completed"}
	cancelledStatuses := []string{"cancelled", "dead"}

	grouped := map[string][]map[string]any{
		"ongoing":   {},
		"completed": {},
		"cancelled": {},
	}

	for _, s := range services {
		status := s.Status
		if status == "dead" {
			status = "cancelled"
		}

		item := map[string]any{
			"id":         s.ID,
			"finalPrice": s.FinalPrice,
			"basePrice":  s.BasePrice,
			"issues":     s.Issues,
			"createdAt":  s.CreatedAt,
			"status":     status,
		}

		switch {
		case contains(ongoingStatuses, s.Status):
			grouped["ongoing"] = append(grouped["ongoing"], item)
		case contains(completedStatuses, s.Status):
			grouped["completed"] = append(grouped["completed"], item)
		case contains(cancelledStatuses, s.Status):
			grouped["cancelled"] = append(grouped["cancelled"], item)
		}
	}

	return grouped, total, nil
}

func contains(arr []string, v string) bool {
	for _, a := range arr {
		if a == v {
			return true
		}
	}
	return false
}

func (s *ProviderService) GetMyService(ctx context.Context, providerID domain.ProviderID, serviceID string) (*domain.AcceptedService, error) {
	return s.AcceptedServiceRepo.FindByIDAndProvider(ctx, serviceID, providerID)
}