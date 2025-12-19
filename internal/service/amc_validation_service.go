package service

import (
	"context"

	"app_backend/internal/repository"
)

type AMCValidationService struct {
	amcRepo *repository.AMCRepo
}

func NewAMCValidationService(
	amcRepo *repository.AMCRepo,
) *AMCValidationService {
	return &AMCValidationService{
		amcRepo: amcRepo,
	}
}

type AMCValidationResult struct {
	AMCAvailable   bool     `json:"amcActive"`
	ValidIssues    []string `json:"amcValidIssues"`
	InvalidIssues  []string `json:"invalidIssues"`
	ProceedWithAMC bool     `json:"canProceedWithAMC"`
	Message        string   `json:"message"`
}

func (s *AMCValidationService) ValidateIssues(
	ctx context.Context,
	vehicleNumber string,
	selectedIssues []string, // 👈 service TYPE names
) (*AMCValidationResult, error) {

	amc, err := s.amcRepo.FindActiveByVehicle(ctx, vehicleNumber)
	if err != nil || amc == nil {
		return &AMCValidationResult{
			AMCAvailable:   false,
			ProceedWithAMC: false,
			Message:        "No active AMC found",
		}, nil
	}

	// Convert AMC valid services to map for O(1)
	covered := make(map[string]bool)
	for _, svc := range amc.ValidServices {
		covered[svc] = true
	}

	var valid []string
	var invalid []string

	for _, issue := range selectedIssues {
		if covered[issue] {
			valid = append(valid, issue)
		} else {
			invalid = append(invalid, issue)
		}
	}

	result := &AMCValidationResult{
		AMCAvailable:   true,
		ValidIssues:    valid,
		InvalidIssues:  invalid,
		ProceedWithAMC: len(invalid) == 0,
	}

	if len(invalid) > 0 {
		result.Message = "AMC not valid for selected problem(s)"
	} else {
		result.Message = "AMC applicable"
	}

	return result, nil
}
