package service

import (
	"context"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationQueryService struct {
	repo *repository.NotificationRepo
}

func NewNotificationQueryService(repo *repository.NotificationRepo) *NotificationQueryService {
	return &NotificationQueryService{repo: repo}
}

func (s *NotificationQueryService) UserServiceNotifications(
	ctx context.Context,
	userID string,
	serviceID string,
	limit, skip int64,
) ([]domain.Notification, error) {

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	sid, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByService(ctx, uid, "user", sid, limit, skip)
}

func (s *NotificationQueryService) ProviderServiceNotifications(
	ctx context.Context,
	providerID string,
	serviceID string,
	limit, skip int64,
) ([]domain.Notification, error) {

	pid, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, err
	}

	sid, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByService(ctx, pid, "provider", sid, limit, skip)
}
