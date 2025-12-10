package service

import (
	"app_backend/internal/domain"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type DashboardResult struct {
	Earnings     []map[string]any `json:"earnings"`
	TotalAmount  float64          `json:"totalAmount"`
	Summary      map[string]any   `json:"summary"`
	FilterConfig map[string]any   `json:"filters"`
}

func (s *ProviderService) GetDashboardStats(ctx context.Context, providerID domain.ProviderID, page, limit int, status, timeframe string) (*DashboardResult, error) {

	filter := bson.M{"provider": providerID}

	if status != "" && status != "all" {
		if status == "pending" {
			filter["paymentStatus"] = "pending"
		} else {
			filter["status"] = status
		}
	}

	if timeframe == "today" {
		start := time.Now()
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		filter["completedAt"] = bson.M{"$gte": start}
	}

	skip := (page - 1) * limit

	services, err := s.AcceptedServiceRepo.Find(ctx, filter, skip, limit)
	if err != nil {
		return nil, err
	}

	totalCompleted, _ := s.AcceptedServiceRepo.Count(ctx, bson.M{
		"provider": providerID,
		"status":   "completed",
	})

	totalCancelled, _ := s.AcceptedServiceRepo.Count(ctx, bson.M{
		"provider": providerID,
		"status":   "cancelled",
	})

	totalPending, _ := s.AcceptedServiceRepo.Count(ctx, bson.M{
		"provider":      providerID,
		"paymentStatus": "pending",
	})

	var totalAmount float64
	earningsArray := make([]map[string]any, len(services))

	for i, s := range services {
		totalAmount += s.FinalPrice

		earningsArray[i] = map[string]any{
			"id":          s.ID,
			"amount":      s.FinalPrice,
			"completedAt": s.CompletedAt,
		}
	}

	return &DashboardResult{
		Earnings:    earningsArray,
		TotalAmount: totalAmount,
		Summary: map[string]any{
			"totalCompleted": totalCompleted,
			"totalCancelled": totalCancelled,
			"totalPending":   totalPending,
			"filteredCount":  len(services),
		},
		FilterConfig: map[string]any{
			"page":      page,
			"limit":     limit,
			"status":    status,
			"timeframe": timeframe,
		},
	}, nil
}
