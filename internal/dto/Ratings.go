package dto

import "time"

type CreateRatingRequest struct {
	BookingID   string `json:"bookingId"`
	Stars       int    `json:"stars"`
	Review      string `json:"review"`
	Recommended bool   `json:"recommended"`
}

type RatingResponse struct {
	ID          string    `json:"id"`
	BookingID   string    `json:"bookingId"`
	RaterID     string    `json:"raterId"`
	RaterName   string    `json:"raterName"`
	RateeID     string    `json:"rateeId"`
	RateeName   string    `json:"rateeName"`
	RatingType  string    `json:"ratingType"`
	Stars       int       `json:"stars"`
	Review      string    `json:"review"`
	Recommended bool      `json:"recommended"`
	CreatedAt   time.Time `json:"createdAt"`
}

type UserRatingsResponse struct {
	AverageRating float64          `json:"average_rating"`
	TotalRatings  int              `json:"total_ratings"`
	Ratings       []RatingResponse `json:"ratings"`
}

type ProviderRatingsResponse struct {
	AverageRating float64          `json:"average_rating"`
	TotalRatings  int              `json:"total_ratings"`
	Ratings       []RatingResponse `json:"ratings"`
}
