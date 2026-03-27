package service

import (
	"app_backend/internal/domain"
	"app_backend/internal/dto"
	"app_backend/internal/repository"
	"context"
	"fmt"
	"strings"
	"time"
)

type AgreementService struct {
	agreementRepo *repository.AgreementRepo
	providerRepo  *repository.ProviderRepo
	kycRepo *repository.KYCRepo
}

func NewAgreementService(ar *repository.AgreementRepo, pr *repository.ProviderRepo, kycRepo *repository.KYCRepo) *AgreementService {
	return &AgreementService{
		agreementRepo: ar,
		providerRepo:  pr,
		kycRepo: kycRepo,
	}
}

func (s *AgreementService) GetProviderAgreement(ctx context.Context, providerID domain.ProviderID) (*dto.AgreementHTMLResponse, error) {

	agreementTemplate, err := s.agreementRepo.FindDefault(ctx)
	if err != nil {
		return nil, fmt.Errorf("agreement template not found: %w", err)
	}

	provider, err := s.providerRepo.FindByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	kyc, err := s.kycRepo.FindByProvider(ctx, string(providerID))
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	var dateTimeStr string
	if provider.AgreementSubmittedAt != nil {
		dateTimeStr = provider.AgreementSubmittedAt.Format("02/01/2006, 15:04")
	} else {
		dateTimeStr = time.Now().Format("02/01/2006, 15:04")
	}

	placeStr := provider.City
	if placeStr == "" {
		placeStr = "New Delhi"
	}

	templateData := map[string]interface{}{
		"provider": map[string]string{
			"name":        kyc.Bank.AccountHolderName,
			"companyName": provider.CompanyName,
		},
		"agreement": map[string]string{
			"dateTime":          dateTimeStr,
			"place":             placeStr,
			"commissionPercent": fmt.Sprintf("%.0f", provider.CommissionPercentage),
		},
	}

	paragraphs := make([]dto.ParagraphHTML, len(agreementTemplate.Paragraphs))
	for i, p := range agreementTemplate.Paragraphs {
		paragraphs[i] = dto.ParagraphHTML{
			Number:  p.Number,
			Title:   s.processTemplate(p.Title, templateData),
			Content: s.processTemplate(p.Content, templateData),
		}
	}

	return &dto.AgreementHTMLResponse{
		ID:                agreementTemplate.ID.Hex(),
		Title:             agreementTemplate.Title,
		AgreementOf:       agreementTemplate.AgreementOf, 
		AgreementFor:      s.processTemplate(agreementTemplate.AgreementFor, templateData),
		CompanyName:       provider.CompanyName,        
		DateTime:          dateTimeStr,                  
		Place:             placeStr,                    
		CommissionPercent: fmt.Sprintf("%.0f", provider.CommissionPercentage), 
		ProviderName:      provider.Name,                
		Paragraphs:        paragraphs,
		CommercialTerms:   agreementTemplate.CommercialTerms,
		MarketplaceFee:    s.processTemplate(agreementTemplate.MarketplaceFee, templateData),  
		PaymentGateway:    agreementTemplate.PaymentGateway,    
		AdditionalNotes:   agreementTemplate.AdditionalNotes,   
		Annexures:         agreementTemplate.Annexures,    
		CreatedAt:         agreementTemplate.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         agreementTemplate.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AgreementService) processTemplate(content string, data map[string]interface{}) string {
	if content == "" {
		return content
	}

	result := content

	if providerData, ok := data["provider"].(map[string]string); ok {
		result = strings.ReplaceAll(result, "{provider.name}", providerData["name"])
		result = strings.ReplaceAll(result, "{provider.companyName}", providerData["companyName"])
	}

	if agreementData, ok := data["agreement"].(map[string]string); ok {
		result = strings.ReplaceAll(result, "{agreement.dateTime}", agreementData["dateTime"])
		result = strings.ReplaceAll(result, "{agreement.place}", agreementData["place"])
		result = strings.ReplaceAll(result, "{agreement.commissionPercent}", agreementData["commissionPercent"])
	}

	return result
}