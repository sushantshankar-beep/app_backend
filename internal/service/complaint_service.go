package service

import (
	"context"
	"time"

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
	req map[string]any,
	raisedBy string,
	authenticatedID string,
) (*domain.Complaint, error) {

	acceptedServiceID, _ := primitive.ObjectIDFromHex(req["acceptedService"].(string))

	photos := []string{}
	if raw, ok := req["photos"].([]any); ok {
		for _, p := range raw {
			if s, ok := p.(string); ok {
				photos = append(photos, s)
			}
		}
	}

	problem := req["problem"].(string)

	var uid, providerID primitive.ObjectID

	if raisedBy == "user" {
		uid, _ = primitive.ObjectIDFromHex(authenticatedID)
		providerID, _ = primitive.ObjectIDFromHex(req["providerId"].(string))
	} else {
		providerID, _ = primitive.ObjectIDFromHex(authenticatedID)
		uid, _ = primitive.ObjectIDFromHex(req["userId"].(string))
	}

	existing, _ := s.repo.FindByAcceptedServiceId(ctx, acceptedServiceID)

	if existing != nil {
		if raisedBy == "user" {
			existing.UserComplaint = &domain.ComplaintSide{
				Problem:  problem,
				Photos:   photos,
				RaisedAt: time.Now(),
			}

			_ = s.acceptedSvcRepo.UpdateComplaintByUser(
				ctx,
				acceptedServiceID,
				existing.ID,
			)
		} else {
			existing.ProviderComplaint = &domain.ComplaintSide{
				Problem:  problem,
				Photos:   photos,
				RaisedAt: time.Now(),
			}

			_ = s.acceptedSvcRepo.UpdateComplaintByProvider(
				ctx,
				acceptedServiceID,
				existing.ID,
			)
		}

		existing.UpdatedAt = time.Now()
		_ = s.repo.Update(ctx, existing)
		return existing, nil
	}

	complaint := &domain.Complaint{
		ID:                primitive.NewObjectID(),
		AcceptedService:   acceptedServiceID,
		ProviderID:        providerID,
		UserID:            uid,
		Status:            "initiated",
		Timeline: map[string]time.Time{
			"initiated": time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if raisedBy == "user" {
		complaint.UserComplaint = &domain.ComplaintSide{
			Problem:  problem,
			Photos:   photos,
			RaisedAt: time.Now(),
		}
	} else {
		complaint.ProviderComplaint = &domain.ComplaintSide{
			Problem:  problem,
			Photos:   photos,
			RaisedAt: time.Now(),
		}
	}

	if err := s.repo.Create(ctx, complaint); err != nil {
		return nil, err
	}

	if raisedBy == "user" {
		_ = s.acceptedSvcRepo.UpdateComplaintByUser(
			ctx,
			acceptedServiceID,
			complaint.ID,
		)
	} else {
		_ = s.acceptedSvcRepo.UpdateComplaintByProvider(
			ctx,
			acceptedServiceID,
			complaint.ID,
		)
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