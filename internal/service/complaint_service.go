package service

import (
	"context"
	"time"
    "strings"
	"errors"
	"fmt"
	"path/filepath"
	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ComplaintService struct {
	repo         *repository.ComplaintRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	acceptedSvcRepo *repository.AcceptedServiceRepo
}

func NewComplaintService(repo *repository.ComplaintRepo, u *repository.UserRepo, p *repository.ProviderRepo,acceptedSvcRepo *repository.AcceptedServiceRepo) *ComplaintService {
	return &ComplaintService{
		repo:         repo,
		userRepo:     u,
		providerRepo: p,
		acceptedSvcRepo: acceptedSvcRepo,
	}
}

func (s *ComplaintService) RaiseComplaint(
	ctx context.Context,
	acceptedServiceStr string,
	problem string,
	photoURLs []string,
	raisedBy string,
	authenticatedID string,
	complaintNumber string,
) (*domain.Complaint, error) {

	acceptedServiceID, err := primitive.ObjectIDFromHex(acceptedServiceStr)
	if err != nil {
		return nil, errors.New("acceptedService is required and must be valid ObjectID")
	}

	if strings.TrimSpace(problem) == "" {
		return nil, errors.New("problem is required")
	}

	acceptedService, err := s.acceptedSvcRepo.FindByID(ctx, acceptedServiceID.Hex())
	if err != nil {
		return nil, errors.New("accepted service not found")
	}

	var (
		userID     primitive.ObjectID
		providerID primitive.ObjectID
	)

	if raisedBy == "user" {
		userID, err = primitive.ObjectIDFromHex(authenticatedID)
		if err != nil {
			return nil, errors.New("invalid authenticated user ID")
		}
		if acceptedService.User != userID {
			return nil, errors.New("unauthorized: this booking does not belong to you")
		}
		providerID = acceptedService.Provider
	} else {
		providerID, err = primitive.ObjectIDFromHex(authenticatedID)
		if err != nil {
			return nil, errors.New("invalid authenticated provider ID")
		}
		if acceptedService.Provider != providerID {
			return nil, errors.New("unauthorized: this booking does not belong to you")
		}
		userID = acceptedService.User
	}

	existing, _ := s.repo.FindByAcceptedServiceId(ctx, acceptedServiceID)

	if existing != nil {
		if raisedBy == "user" && existing.UserComplaint != nil {
			return nil, errors.New("you have already raised a complaint for this booking")
		}
		if raisedBy == "provider" && existing.ProviderComplaint != nil {
			return nil, errors.New("you have already raised a complaint for this booking")
		}

		side := &domain.ComplaintSide{
			Problem:  problem,
			Photos:   photoURLs,
			RaisedAt: time.Now(),
		}

		if raisedBy == "user" {
			existing.UserComplaint = side
			_ = s.acceptedSvcRepo.UpdateComplaintByUser(ctx, acceptedServiceID, existing.ID)
		} else {
			existing.ProviderComplaint = side
			_ = s.acceptedSvcRepo.UpdateComplaintByProvider(ctx, acceptedServiceID, existing.ID)
		}

		existing.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, existing)
		return existing, nil
	}

	complaint := &domain.Complaint{
		ID:              primitive.NewObjectID(),
		ComplaintNumber: complaintNumber,
		AcceptedService: acceptedServiceID,
		ProviderID:      providerID,
		UserID:          userID,
		Status:          "initiated",
		Timeline: map[string]time.Time{
			"initiated": time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	side := &domain.ComplaintSide{
		Problem:  problem,
		Photos:   photoURLs,
		RaisedAt: time.Now(),
	}

	if raisedBy == "user" {
		complaint.UserComplaint = side
	} else {
		complaint.ProviderComplaint = side
	}

	if err := s.repo.Create(ctx, complaint); err != nil {
		return nil, err
	}

	if raisedBy == "user" {
		_ = s.acceptedSvcRepo.UpdateComplaintByUser(ctx, acceptedServiceID, complaint.ID)
	} else {
		_ = s.acceptedSvcRepo.UpdateComplaintByProvider(ctx, acceptedServiceID, complaint.ID)
	}

	return complaint, nil
}

func (s *ComplaintService) GetUserComplaints(ctx context.Context, uid string) ([]domain.Complaint, error) {
	id, _ := primitive.ObjectIDFromHex(uid)
	return s.repo.FindByUser(ctx, id)
}

func (s *ComplaintService) GetProviderComplaints(ctx context.Context, pid string) ([]domain.Complaint, error) {
	id, _ := primitive.ObjectIDFromHex(pid)
	return s.repo.FindByProvider(ctx, id)
}

func getString(req map[string]any, key string) (string, bool) {
	v, ok := req[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok && s != ""
}

func getObjectID(req map[string]any, key string) (primitive.ObjectID, error) {
	s, ok := getString(req, key)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("%s is required", key)
	}
	id, err := primitive.ObjectIDFromHex(s)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("%s must be a valid ObjectID", key)
	}
	return id, nil
}

func validateImages(raw any) ([]string, error) {
	if raw == nil {
		return []string{}, nil
	}

	arr, ok := raw.([]any)
	if !ok {
		return nil, errors.New("photos must be an array")
	}

	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	}

	var photos []string
	for i, p := range arr {
		s, ok := p.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("photo at index %d must be a string", i)
		}

		ext := strings.ToLower(filepath.Ext(s))
		if !allowed[ext] {
			return nil, fmt.Errorf("invalid image format at index %d", i)
		}

		photos = append(photos, s)
	}

	return photos, nil
}

func (s *ComplaintService) GetNextComplaintSequence(ctx context.Context) (int64, error) {
    return s.repo.GetNextSequence(ctx, "complaint")
}