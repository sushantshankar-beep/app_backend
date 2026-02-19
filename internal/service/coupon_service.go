package service

import (
	"context"
	"errors"
	"app_backend/internal/dto"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
    "time"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CouponService struct {
	promoRepo           *repository.PromoRepo
	discountRepo        *repository.DiscountRepo
	acceptedServiceRepo ports.AcceptedServiceRepository
}

func NewCouponService(
	promoRepo *repository.PromoRepo,
	discountRepo *repository.DiscountRepo,
	acceptedServiceRepo ports.AcceptedServiceRepository,
) *CouponService {
	return &CouponService{
		promoRepo:           promoRepo,
		discountRepo: discountRepo,
		acceptedServiceRepo: acceptedServiceRepo,
	}
}

func (s *CouponService) GetAvailableCoupons(
	ctx context.Context,
	serviceID string,
	page, limit int,
) ([]dto.PromoCodeListResponse, int64, error) {

	svcOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, 0, errors.New("invalid serviceId")
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, svcOID)
	if err != nil {
		return nil, 0, errors.New("service not found")
	}

	promos, total, err := s.promoRepo.GetActiveForService(
		ctx,
		svc.ServiceType,
		page,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}

	var response []dto.PromoCodeListResponse

	for _, p := range promos {
		response = append(response, dto.PromoCodeListResponse{
			ID:           p.ID.Hex(),
			Code:         p.Code,
			Title:        p.Title,
			ServiceType:  svc.ServiceType,
			Status:       p.Status,
			CreatedBy:    p.CreatedBy,
			ValidityStart: p.StartAt.Format(time.RFC3339),
			ValidityEnd: p.EndAt.Format(time.RFC3339),
			CreatedAt:    p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
		})
	}

	return response, total, nil
}

func (s *CouponService) ValidateAndApply(
	ctx context.Context,
	userID string,
	code string,
	serviceID string,
	isNewUser bool,
) (*domain.CouponApplyResult, error) {

	svcOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, errors.New("invalid serviceId")
	}
	svc, err := s.acceptedServiceRepo.GetByID(ctx, svcOID)
	if err != nil {
		return nil, errors.New("service not found")
	}

	amount := svc.FinalPrice
	serviceType := "All Services"

	result := &domain.CouponApplyResult{
		OriginalAmount:   amount,
		DiscountedAmount: amount,
	}

	discounts, err := s.discountRepo.GetActiveForService(ctx, serviceType, "","")
	if err == nil {
		for _, d := range discounts {
			if !discountUserEligible(d.UserEligibility, isNewUser) {
				continue
			}
			disc := calcDiscount(d.Type, d.Value, d.MaxDiscount, result.DiscountedAmount)
			if disc > 0 {
				result.TotalDiscount += disc
				result.DiscountedAmount -= disc
				result.AppliedDiscount = &domain.AppliedDiscountSummary{
					DiscountID:  d.ID.Hex(),
					Name:        d.Name,
					DiscountAmt: disc,
				}
				break
			}
		}
	}

	if code == "" {
		return result, nil
	}

	promo, err := s.promoRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, errors.New("invalid promo code")
	}

	now := time.Now()
	if promo.Status != domain.PromoStatusActive {
		return nil, errors.New("promo code is not active")
	}
	if promo.EndAt != nil && now.After(*promo.EndAt) {
		return nil, errors.New("promo code has expired")
	}
	if now.Before(promo.StartAt) {
		return nil, errors.New("promo code is not yet valid")
	}
	if !promoUserEligible(promo.UserEligibility, isNewUser) {
		return nil, errors.New("you are not eligible for this promo")
	}
	if promo.MinOrderValue > 0 && amount < promo.MinOrderValue {
		return nil, errors.New("order value too low for this promo")
	}

	validService := false
	for _, st := range promo.ServiceTypes {
		if st == domain.PromoServiceAll || string(st) == serviceType {
			validService = true
			break
		}
	}
	if !validService {
		return nil, errors.New("promo not valid for this service type")
	}

	if result.AppliedDiscount != nil && !promo.AllowStacking {
		result.TotalDiscount -= result.AppliedDiscount.DiscountAmt
		result.DiscountedAmount = result.OriginalAmount
		result.AppliedDiscount = nil
	}

	disc := calcDiscount(promo.DiscountType, promo.Value, promo.MaxDiscount, result.DiscountedAmount)
	result.TotalDiscount += disc
	result.DiscountedAmount -= disc
	result.AppliedPromo = &domain.AppliedPromoSummary{
		PromoID:     promo.ID.Hex(),
		Code:        promo.Code,
		DiscountAmt: disc,
	}

	return result, nil
}

func calcDiscount(dtype domain.DiscountType, value, maxDiscount, amount float64) float64 {
	var disc float64
	if dtype == domain.DiscountPercent {
		disc = amount * value / 100
		if maxDiscount > 0 && disc > maxDiscount {
			disc = maxDiscount
		}
	} else {
		disc = value
	}
	if disc > amount {
		disc = amount
	}
	return disc
}


func promoUserEligible(eligibility domain.UserEligibility, isNewUser bool) bool {
	switch eligibility {
	case domain.UserEligibilityNew:
		return isNewUser
	case domain.UserEligibilityExisting:
		return !isNewUser
	default:
		return true
	}
}

func discountUserEligible(eligibility domain.UserEligibility, isNewUser bool) bool {
	switch eligibility {
	case domain.UserEligibilityNew:
		return isNewUser
	case domain.UserEligibilityExisting:
		return !isNewUser
	default:
		return true
	}
}

func (s *CouponService) ConfirmPromoUsage(ctx context.Context, promoID string) error {
	oid, err := primitive.ObjectIDFromHex(promoID)
	if err != nil {
		return err
	}
	return s.promoRepo.IncrementUsage(ctx, oid)
}