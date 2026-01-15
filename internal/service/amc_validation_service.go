package service

import (
	"context"

	"app_backend/internal/repository"
)

type AMCValidationService struct {
	amcRepo *repository.AMCRepo
}

func NewAMCValidationService(amcRepo *repository.AMCRepo) *AMCValidationService {
	return &AMCValidationService{amcRepo: amcRepo}
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
	selectedIssues []string,
) (*AMCValidationResult, error) {

	amc, err := s.amcRepo.FindActiveByVehicle(ctx, vehicleNumber)
	if err != nil || amc == nil {
		return &AMCValidationResult{
			AMCAvailable:   false,
			ProceedWithAMC: false,
			Message:        "No active AMC found",
		}, nil
	}

	covered := make(map[string]bool, len(amc.ValidServices))
	for _, svc := range amc.ValidServices {
		covered[svc] = true
	}

	var valid, invalid []string
	for _, issue := range selectedIssues {
		if covered[issue] {
			valid = append(valid, issue)
		} else {
			invalid = append(invalid, issue)
		}
	}

	return &AMCValidationResult{
		AMCAvailable:   true,
		ValidIssues:    valid,
		InvalidIssues:  invalid,
		ProceedWithAMC: len(invalid) == 0,
		Message: func() string {
			if len(invalid) > 0 {
				return "AMC not valid for selected problem(s)"
			}
			return "AMC applicable"
		}(),
	}, nil
}
