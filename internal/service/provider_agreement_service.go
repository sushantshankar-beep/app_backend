package service

import (
	"time"
	"context"
	"html/template"
	"app_backend/internal/dto"
	"app_backend/internal/repository"
)

type AgreementService struct {
	agreementRepo *repository.AgreementRepo
}

func NewAgreementService(ar *repository.AgreementRepo) *AgreementService {
	return &AgreementService{
		agreementRepo: ar,
	}
}

func (s *AgreementService) GetAgreement(ctx context.Context, id string) (*dto.AgreementResponse, error) {
	a, err := s.agreementRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	paragraphs := make([]dto.ParagraphResponse, len(a.Paragraphs))
	for i, p := range a.Paragraphs {
		paragraphs[i] = dto.ParagraphResponse{
			Number:  p.Number,
			Title:   p.Title,
			Content: p.Content,
		}
	}

	return &dto.AgreementResponse{
		ID:              a.ID.Hex(),
		Title:           a.Title,
		Paragraphs:      paragraphs,
		AgreementOf:     a.AgreementOf,
		AgreementFor:    a.AgreementFor,
		CompanyName:     a.CompanyName,
		Annexures:       a.Annexures,
		CommercialTerms: a.CommercialTerms,
		MarketplaceFee:  a.MarketplaceFee,
		PaymentGateway:  a.PaymentGateway,
		AdditionalNotes: a.AdditionalNotes,
		CreatedAt:       a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       a.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AgreementService) GetAgreementSafeHTML(ctx context.Context, id string) (*dto.SafeHTMLAgreementResponse, error) {
	a, err := s.agreementRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	paragraphs := make([]dto.SafeParagraphResponse, len(a.Paragraphs))
	for i, p := range a.Paragraphs {
		paragraphs[i] = dto.SafeParagraphResponse{
			Number:  p.Number,
			Title:   p.Title,
			Content: template.HTML(p.Content),
		}
	}

	return &dto.SafeHTMLAgreementResponse{
		ID:              a.ID.Hex(),
		Title:           a.Title,
		Paragraphs:      paragraphs,
		AgreementOf:     template.HTML(a.AgreementOf),
		AgreementFor:    template.HTML(a.AgreementFor),
		CompanyName:     template.HTML(a.CompanyName),
		Annexures:       template.HTML(a.Annexures),
		CommercialTerms: template.HTML(a.CommercialTerms),
		MarketplaceFee:  template.HTML(a.MarketplaceFee),
		PaymentGateway:  template.HTML(a.PaymentGateway),
		AdditionalNotes: template.HTML(a.AdditionalNotes),
		CreatedAt:       a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       a.UpdatedAt.Format(time.RFC3339),
	}, nil
}