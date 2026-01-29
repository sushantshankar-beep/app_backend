package dto

import 	"html/template"

type ParagraphResponse struct {
	Number  string `json:"number"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type AgreementResponse struct {
	ID              string              `json:"_id"`
	Title           string              `json:"title"`
	Paragraphs      []ParagraphResponse `json:"paragraphs"`
	AgreementOf     string              `json:"agreementOf"`
	AgreementFor    string              `json:"agreementFor"`
	CompanyName     string              `json:"companyName"`
	Annexures       string              `json:"annexures"`
	CommercialTerms string              `json:"commercialTerms"`
	MarketplaceFee  string              `json:"marketplaceFee"`
	PaymentGateway  string              `json:"paymentGateway"`
	AdditionalNotes string              `json:"additionalNotes"`
	CreatedAt       string              `json:"createdAt"`
	UpdatedAt       string              `json:"updatedAt"`
}

type SafeHTMLAgreementResponse struct {
	ID              string                  `json:"_id"`
	Title           string                  `json:"title"`
	Paragraphs      []SafeParagraphResponse `json:"paragraphs"`
	AgreementOf     template.HTML           `json:"agreementOf"`
	AgreementFor    template.HTML           `json:"agreementFor"`
	CompanyName     template.HTML           `json:"companyName"`
	Annexures       template.HTML           `json:"annexures"`
	CommercialTerms template.HTML           `json:"commercialTerms"`
	MarketplaceFee  template.HTML           `json:"marketplaceFee"`
	PaymentGateway  template.HTML           `json:"paymentGateway"`
	AdditionalNotes template.HTML           `json:"additionalNotes"`
	CreatedAt       string                  `json:"createdAt"`
	UpdatedAt       string                  `json:"updatedAt"`
}

type SafeParagraphResponse struct {
	Number  string        `json:"number"`
	Title   string        `json:"title"`
	Content template.HTML `json:"content"`
}
