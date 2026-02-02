package dto

type ProviderServicesAndReviewsResponse struct {
	Services      []string         `json:"services"`
	AverageRating float64          `json:"averageRating"`
	TotalRatings  int              `json:"totalRatings"`
	TopReviews    []RatingResponse `json:"topReviews"`
}