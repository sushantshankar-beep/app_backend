package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/validation"
	"app_backend/internal/worker"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProviderService struct {
    repo                ports.ProviderRepository
    otp                 ports.OTPStore
    token               ports.TokenService
    queue               *worker.OTPQueue
    AcceptedServiceRepo ports.AcceptedServiceRepository
    ServiceRequestRepo  ports.ServiceRequestRepository
    UserRepo            ports.UserRepository
    BidRepo             ports.BidRepository
}


func NewProviderService(
    repo ports.ProviderRepository,
    otp ports.OTPStore,
    token ports.TokenService,
    q *worker.OTPQueue,
    acceptedRepo ports.AcceptedServiceRepository,
    srRepo ports.ServiceRequestRepository,
    userRepo ports.UserRepository,
    bidRepo ports.BidRepository,
) *ProviderService {
    return &ProviderService{
        repo:                repo,
        otp:                 otp,
        token:               token,
        queue:               q,
        AcceptedServiceRepo: acceptedRepo,
        ServiceRequestRepo:  srRepo,
        UserRepo:            userRepo,
        BidRepo:             bidRepo,
    }
}
func firstOrFallback(arr []string, fallback string) string {
    if len(arr) > 0 {
        return arr[0]
    }
    return fallback
}

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

func (s *ProviderService) CreateOrUpdateProfile(ctx context.Context, id domain.ProviderID, req validation.ProviderProfileRequest) (*domain.Provider, error) {
	provider, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		provider.Name = req.Name
	}
	if req.Email != "" {
		provider.Email = req.Email
	}
	if req.AlternateContact != "" {
		provider.AlternateContact = req.AlternateContact
	}
	if req.ProfileURL != "" {
		provider.ProfileURL = req.ProfileURL
	}
	if req.Address != "" {
		provider.Address = req.Address
	}
	if req.PermanentAddress != "" {
		provider.PermanentAddress = req.PermanentAddress
	}
	if req.City != "" {
		provider.City = req.City
	}
	if req.GSTNumber != "" {
		provider.GSTNumber = req.GSTNumber
	}
	if req.VehicleNumber != "" {
		provider.VehicleNumber = req.VehicleNumber
	}
	if req.Description != "" {
		provider.Description = req.Description
	}
	if req.CompanyName != "" {
		provider.CompanyName = req.CompanyName
	}
	if req.PreferredLanguage != "" {
		provider.PreferredLanguage = req.PreferredLanguage
	}

	provider.TermsAndConditions = req.TermsAndConditions

	if len(req.VehicleType) > 0 {
		provider.VehicleType = req.VehicleType
	}

	if len(req.ProviderServices) > 0 {
		provider.ProviderServices = req.ProviderServices
	}

	if len(req.ProviderBrands) > 0 {
		provider.ProviderBrands = req.ProviderBrands
	}

	// if len(req.IdentityProof) > 0 {
	// 	provider.IdentityProof = make([]domain.Proof, len(req.IdentityProof))
	// 	for i, ip := range req.IdentityProof {
	// 		provider.IdentityProof[i] = domain.Proof{
	// 			Type:     ip.Type,
	// 			File:     ip.File,
	// 			Verified: "pending",
	// 		}
	// 	}
	// }

	// if len(req.AddressProof) > 0 {
	// 	provider.AddressProof = make([]domain.Proof, len(req.AddressProof))
	// 	for i, ap := range req.AddressProof {
	// 		provider.AddressProof[i] = domain.Proof{
	// 			Type:     ap.Type,
	// 			File:     ap.File,
	// 			Verified: "pending",
	// 		}
	// 	}
	// }

	// if req.CancelCheque != nil && req.CancelCheque.File != "" {
	// 	provider.CancelCheque = domain.CancelCheque{
	// 		File:     req.CancelCheque.File,
	// 		Verified: "pending",
	// 	}
	// }

	if req.BankDetails != nil {
		if req.BankDetails.AccountHolderName != "" {
			provider.BankDetails.AccountHolderName = req.BankDetails.AccountHolderName
		}
		if req.BankDetails.AccountNumber != "" {
			provider.BankDetails.AccountNumber = req.BankDetails.AccountNumber
		}
		if req.BankDetails.IFSCCode != "" {
			provider.BankDetails.IFSCCode = req.BankDetails.IFSCCode
		}
		if req.BankDetails.BranchName != "" {
			provider.BankDetails.BranchName = req.BankDetails.BranchName
		}
		if req.BankDetails.UPI != "" {
			provider.BankDetails.UPI = req.BankDetails.UPI
		}
	}

	provider.FormSubmitted++
	provider.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *ProviderService) GetMyAllServices(
    ctx context.Context,
    providerID domain.ProviderID,
    page, limit int,
) (map[string][]map[string]any, int64, error) {

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

    for _, svc := range services {

        // ---- Fetch Service Request ----
        sr, err := s.ServiceRequestRepo.FindByObjectID(ctx, svc.ServiceRequest)
        if err != nil {
            return nil, 0, err
        }

        // ---- Fetch User ----
        userData, err := s.UserRepo.FindByObjectID(ctx, sr.User)
        if err != nil {
            return nil, 0, err
        }

        // ---- Fetch Provider ----
        providerData, err := s.repo.FindByID(ctx, providerID)
        if err != nil {
            return nil, 0, err
        }

        // ---- Fetch accepted bid ----
        bid, err := s.BidRepo.FindByObjectID(ctx, svc.AcceptedBid)
        if err != nil {
            return nil, 0, err
        }

        // ---- Build Response ----
        item := map[string]any{
            "id":  svc.ID,
            "_id": svc.ID,

            "user": map[string]any{
                "id":         userData.ID,
                "name":       userData.Name,
                "email":      userData.Email,
                "phone":      userData.Phone,
            },

            "provider": map[string]any{
                "id":          providerData.ID,
                "name":        providerData.Name,
                "email":       providerData.Email,
                "phone":       providerData.Phone,
                "businessName": providerData.CompanyName,
            },

            "serviceRequest": map[string]any{
                "id":            sr.ID,
                "vehicleNumber": sr.VehicleNumber,
                "vehicleModel":  sr.Model,
                "vehicleBrand":  sr.Brand,
                "vehicleYear":   sr.Year,
                "vehicleType":   sr.VehicleType,
                "fuelType":      sr.FuelType,
                "location":      sr.Address,
                "serviceType":   firstOrFallback(sr.Problems, sr.ServiceType),
                "issues":        svc.Issues,
                "description":   sr.Description,
                "scheduledDate": sr.ScheduledDate,
            },

            "acceptedBid": map[string]any{
                "amount":        bid.OfferedPrice,
                "basePrice":     bid.BasePrice,
                "estimatedTime": bid.EstimatedTime,
                "distance":      bid.Distance,
                "message":       bid.Message,
                "createdAt":     bid.CreatedAt,
            },

            "otp": map[string]any{
                "code":       svc.OTP.Code,
                "verified":   svc.OTP.Verified,
                "verifiedAt": svc.OTP.VerifiedAt,
            },

            "status":        svc.Status,
            "reachedAt":     svc.ReachedAt,
            "startedAt":     svc.StartedAt,
            "completedAt":   svc.CompletedAt,
            "expiresAt":     svc.ExpiresAt,
            "basePrice":     svc.BasePrice,
            "finalPrice":    svc.FinalPrice,
            "paymentStatus": svc.PaymentStatus,
            "orderId":       svc.OrderID,

            "serviceLocation": svc.ServiceLocation,

            "createdAt": svc.CreatedAt,
            "updatedAt": svc.UpdatedAt,
        }

        switch {
        case contains(ongoingStatuses, svc.Status):
            grouped["ongoing"] = append(grouped["ongoing"], item)
        case contains(completedStatuses, svc.Status):
            grouped["completed"] = append(grouped["completed"], item)
        case contains(cancelledStatuses, svc.Status):
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


func (s *ProviderService) GetDashboardStats(ctx context.Context, providerID domain.ProviderID) (*domain.DashboardStats, error) {
	startOfDay := time.Now().Truncate(24 * time.Hour)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"provider": providerID, "paymentStatus": "paid"}}},
		{{
			Key: "$facet",
			Value: bson.M{
				"allTimeEarning": []bson.M{
					{"$match": bson.M{"status": "completed"}},
					{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$finalPrice"}}},
				},
				"todaysEarning": []bson.M{
					{"$match": bson.M{"status": "completed", "updatedAt": bson.M{"$gte": startOfDay}}},
					{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$finalPrice"}}},
				},
				"servicesCompleted": []bson.M{
					{"$match": bson.M{"status": "completed"}},
					{"$count": "count"},
				},
				"cancelledServices": []bson.M{
					{"$match": bson.M{"status": "cancelled"}},
					{"$count": "count"},
				},
			},
		}},
	}

	resultArr, err := s.AcceptedServiceRepo.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var stats domain.DashboardStats
	if len(resultArr) > 0 {
		result := resultArr[0]

		if allTime, ok := result["allTimeEarning"].([]interface{}); ok && len(allTime) > 0 {
			if v, ok := allTime[0].(map[string]interface{})["total"].(float64); ok {
				stats.AllTimeEarning = v
			}
		}

		if today, ok := result["todaysEarning"].([]interface{}); ok && len(today) > 0 {
			if v, ok := today[0].(map[string]interface{})["total"].(float64); ok {
				stats.TodaysEarning = v
			}
		}

		if completed, ok := result["servicesCompleted"].([]interface{}); ok && len(completed) > 0 {
			if v, ok := completed[0].(map[string]interface{})["count"].(int32); ok {
				stats.ServicesCompleted = int(v)
			} else if v, ok := completed[0].(map[string]interface{})["count"].(int64); ok {
				stats.ServicesCompleted = int(v)
			}
		}

		if cancelled, ok := result["cancelledServices"].([]interface{}); ok && len(cancelled) > 0 {
			if v, ok := cancelled[0].(map[string]interface{})["count"].(int32); ok {
				stats.CancelledServices = int(v)
			} else if v, ok := cancelled[0].(map[string]interface{})["count"].(int64); ok {
				stats.CancelledServices = int(v)
			}
		}
	}

	return &stats, nil
}
