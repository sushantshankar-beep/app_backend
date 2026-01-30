package dto

type AgreementHTMLResponse struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	AgreementOf     string         `json:"agreementOf"`
	AgreementFor    string         `json:"agreementFor"`
	CompanyName     string         `json:"companyName"`
	DateTime        string         `json:"dateTime"`
	CommissionPercent string        `json:commissionPercent`
	Place           string         `json:"place"`
	Paragraphs      []ParagraphHTML `json:"paragraphs"`
	CommercialTerms string         `json:"commercialTerms"`
	MarketplaceFee  string         `json:"marketplaceFee"`
	PaymentGateway  string         `json:"paymentGateway"`
	AdditionalNotes string         `json:"additionalNotes"`
	ProviderName       string       `json:"providerName"`
	Annexures       string         `json:"annexures"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
}

type ParagraphHTML struct {
	Number  string `json:"number"`
	Title   string `json:"title"`
	Content string `json:"content"`
}