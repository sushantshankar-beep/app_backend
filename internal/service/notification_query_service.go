package service

import (
	"context"
	"errors"

	"app_backend/internal/dto"
	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationQueryService struct {
	repo *repository.NotificationRepo
}

func NewNotificationQueryService(
	repo *repository.NotificationRepo,
) *NotificationQueryService {
	return &NotificationQueryService{repo: repo}
}

/* ============================================================
   LATEST SERVICE NOTIFICATIONS ✅ NEW
============================================================ */

func (s *NotificationQueryService) ListLatestServiceNotifications(
	ctx context.Context,
	ownerID string,
	ownerType string,
	limit int64,
	skip int64,
) (primitive.ObjectID, []dto.NotificationResponse, error) {

	oid, err := primitive.ObjectIDFromHex(ownerID)
	if err != nil {
		return primitive.NilObjectID, nil, errors.New("invalid owner id")
	}

	serviceID, err := s.repo.FindLatestServiceForOwner(
		ctx,
		oid,
		ownerType,
	)
	if err != nil {
		return primitive.NilObjectID, nil, err
	}

	items, err := s.repo.FindByService(
		ctx,
		oid,
		ownerType,
		serviceID,
		limit,
		skip,
	)

	if err != nil {
		return primitive.NilObjectID, nil, err
	}

	out := make([]dto.NotificationResponse, 0, len(items))

	for _, n := range items {
		out = append(out, mapNotification(n))
	}

	return serviceID, out, nil
}

/* ============================================================
   MAPPER
============================================================ */

func mapNotification(n domain.Notification) dto.NotificationResponse {

	return dto.NotificationResponse{
		ID:        n.ID.Hex(),
		OwnerID:  n.OwnerID.Hex(),
		OwnerType: n.OwnerType,
		ServiceID: n.ServiceID.Hex(),

		Title: n.Title,
		Body:  n.Body,
		Data:  n.Data,

		Read:   n.Read,
		Status: n.Status,

		CreatedAt: n.CreatedAt,
	}
}
