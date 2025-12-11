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
}

func NewComplaintService(repo *repository.ComplaintRepo, u *repository.UserRepo, p *repository.ProviderRepo) *ComplaintService {
    return &ComplaintService{
        repo: repo,
        userRepo: u,
        providerRepo: p,
    }
}

func (s *ComplaintService) RaiseComplaint(ctx context.Context, req map[string]any, raisedBy string, userID string) (*domain.Complaint, error) {

    providerID, _ := primitive.ObjectIDFromHex(req["providerId"].(string))
    acceptedService, _ := primitive.ObjectIDFromHex(req["acceptedService"].(string))

    uid, _ := primitive.ObjectIDFromHex(userID)

    complaint := &domain.Complaint{
        ID: primitive.NewObjectID(),
        AcceptedService: acceptedService,
        AcceptedServiceId: req["acceptedServiceId"].(int64),
        ProviderID: providerID,
        UserID: uid,
        RaisedBy: raisedBy,
        Problem: req["problem"].(string),
        Photos: req["photos"].([]string),
        Status: "initiated",
        Timeline: map[string]time.Time{
            "initiated": time.Now(),
        },
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    if err := s.repo.Create(ctx, complaint); err != nil {
        return nil, err
    }
    _ = s.userRepo.AddComplaint(ctx, uid, complaint.ID)
    _ = s.providerRepo.AddComplaint(ctx, providerID, complaint.ID)

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
