package service

import (
	"context"
	"errors"

	"app_backend/internal/domain"
	"app_backend/internal/dto"
	"app_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RatingService struct {
	ratingRepo   *repository.RatingRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	bookingRepo  *repository.AcceptedServiceRepo
}

func NewRatingService(
	ratingRepo *repository.RatingRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	bookingRepo *repository.AcceptedServiceRepo,
) *RatingService {
	return &RatingService{
		ratingRepo:   ratingRepo,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		bookingRepo:  bookingRepo,
	}
}

func (s *RatingService) CreateUserRating(ctx context.Context, userID string, req dto.CreateRatingRequest) (*dto.RatingResponse, error) {

	bookingObjID, err := primitive.ObjectIDFromHex(req.BookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.bookingRepo.GetByID(ctx, bookingObjID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	providerObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid provider ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(booking.User.Hex())
	if err != nil {
		return nil, errors.New("invalid user ID in booking")
	}

	existingRating, err := s.ratingRepo.GetByBookingAndRater(ctx, bookingObjID, providerObjID)
	if err == nil && existingRating != nil {
		return nil, errors.New("rating already submitted for this booking")
	}

	rating := &domain.Rating{
		BookingID:     bookingObjID,
		RaterID:       providerObjID,
		RateeID:       userObjID,
		RatingType:    domain.RATING_USER,
		Stars:         req.Stars,
		Review:        req.Review,
		Recommended:   req.Recommended,
	}

	if err := s.ratingRepo.Create(ctx, rating); err != nil {
		return nil, err
	}

	user, _ := s.userRepo.GetByID(ctx, userObjID)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))

	return &dto.RatingResponse{
		ID:            rating.ID.Hex(),
		BookingID:     rating.BookingID.Hex(),
		RaterID:       rating.RaterID.Hex(),
		RaterName:     provider.Name,
		RateeID:       rating.RateeID.Hex(),
		RateeName:     user.Name,
		RatingType:    string(rating.RatingType),
		Stars:         rating.Stars,
		Review:        rating.Review,
		Recommended:   *rating.Recommended,
		CreatedAt:     rating.CreatedAt,
	}, nil
}

func (s *RatingService) CreateProviderRating(ctx context.Context, userID string, req dto.CreateRatingRequest) (*dto.RatingResponse, error) {

	bookingObjID, err := primitive.ObjectIDFromHex(req.BookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.bookingRepo.GetByID(ctx, bookingObjID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	providerObjID, err := primitive.ObjectIDFromHex(booking.Provider.Hex())
	if err != nil {
		return nil, errors.New("invalid provider ID in booking")
	}

	existingRating, err := s.ratingRepo.GetByBookingAndRater(ctx, bookingObjID, userObjID)
	if err == nil && existingRating != nil {
		return nil, errors.New("rating already submitted for this booking")
	}

	rating := &domain.Rating{
		BookingID:     bookingObjID,
		RaterID:       userObjID,
		RateeID:       providerObjID,
		RatingType:    domain.RATING_PROVIDER,
		Stars:         req.Stars,
		Review:        req.Review,
		Recommended:   req.Recommended,
	}

	if err := s.ratingRepo.Create(ctx, rating); err != nil {
		return nil, err
	}

	user, _ := s.userRepo.GetByID(ctx, userObjID)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(providerObjID.Hex()))

	return &dto.RatingResponse{
		ID:            rating.ID.Hex(),
		BookingID:     rating.BookingID.Hex(),
		RaterID:       rating.RaterID.Hex(),
		RaterName:     user.Name,
		RateeID:       rating.RateeID.Hex(),
		RateeName:     provider.Name,
		RatingType:    string(rating.RatingType),
		Stars:         rating.Stars,
		Review:        rating.Review,
		Recommended:   *rating.Recommended,
		CreatedAt:     rating.CreatedAt,
	}, nil
}

func (s *RatingService) GetUserRatings(ctx context.Context, userID string) (*dto.UserRatingsResponse, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	ratings, err := s.ratingRepo.GetRatingsByRatee(ctx, userObjID, domain.RATING_USER)
	if err != nil {
		return nil, err
	}

	var total int
	for _, r := range ratings {
		total += int(r.Stars)
	}

	avg := 0.0
	if len(ratings) > 0 {
		avg = float64(total) / float64(len(ratings))
	}

	mappedRatings, err := s.mapRatingsToResponse(ctx, ratings)
	if err != nil {
		return nil, err
	}

	return &dto.UserRatingsResponse{
		AverageRating: avg,
		TotalRatings:  len(ratings),
		Ratings:       mappedRatings,
	}, nil
}


func (s *RatingService) GetProviderRatings(
	ctx context.Context,
	providerID string,
) (*dto.ProviderRatingsResponse, error) {

	providerObjID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, errors.New("invalid provider ID")
	}

	ratings, err := s.ratingRepo.GetRatingsByRatee(
		ctx,
		providerObjID,
		domain.RATING_PROVIDER,
	)
	if err != nil {
		return nil, err
	}

	var total int
	for _, r := range ratings {
		total += int(r.Stars)
	}

	avg := 0.0
	if len(ratings) > 0 {
		avg = float64(total) / float64(len(ratings))
	}

	mapped, err := s.mapRatingsToResponse(ctx, ratings)
	if err != nil {
		return nil, err
	}

	return &dto.ProviderRatingsResponse{
		AverageRating: avg,
		TotalRatings:  len(ratings),
		Ratings:       mapped,
	}, nil
}


func (s *RatingService) mapRatingsToResponse(ctx context.Context, ratings []domain.Rating) ([]dto.RatingResponse, error) {
	result := make([]dto.RatingResponse, 0, len(ratings))

	for _, r := range ratings {
		var raterName, rateeName string

		if r.RatingType == domain.RATING_USER {
			if provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(r.RaterID.Hex())); err == nil && provider != nil {
				raterName = provider.Name
			}

			if user, err := s.userRepo.GetByID(ctx, r.RateeID); err == nil && user != nil {
				rateeName = user.Name
			}
		} else {
			if user, err := s.userRepo.GetByID(ctx, r.RaterID); err == nil && user != nil {
				raterName = user.Name
			}

			if provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(r.RateeID.Hex())); err == nil && provider != nil {
				rateeName = provider.Name
			}
		}

		result = append(result, dto.RatingResponse{
			ID:          r.ID.Hex(),
			BookingID:   r.BookingID.Hex(),
			RaterID:     r.RaterID.Hex(),
			RaterName:   raterName,
			RateeID:     r.RateeID.Hex(),
			RateeName:   rateeName,
			RatingType:  string(r.RatingType),
			Stars:       r.Stars,
			Review:      r.Review,
			Recommended: *r.Recommended,
			CreatedAt:   r.CreatedAt,
		})
	}

	return result, nil
}