package validation

import (
	"errors"
	"strings"
)

type HomepageRequest struct {
	ID         string                 `json:"id"`
	UserCohort string                 `json:"userCohort" binding:"required"`
	Location   LocationRequest        `json:"location" binding:"required"`
	Banners    []BannerRequest        `json:"banners"`
	Categories []CategoryRequest      `json:"categories"`
	IsActive   bool                   `json:"isActive"`
}

type LocationRequest struct {
	City             string  `json:"city" binding:"required"`
	Address          string  `json:"address"`
	Latitude         float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude        float64 `json:"longitude" binding:"required,min=-180,max=180"`
	Pincode          string  `json:"pincode"`
	FormattedAddress string  `json:"formattedAddress"`
}

type BannerRequest struct {
	Title             string         `json:"title" binding:"required"`
	ImageURL          string         `json:"imageUrl" binding:"required,url"`
	RedirectionURL    string         `json:"redirectionUrl" binding:"required"`
	RedirectionParams map[string]any `json:"redirectionParams"`
}

type CategoryRequest struct {
	Name           string `json:"name" binding:"required"`
	IconURL        string `json:"iconUrl" binding:"required,url"`
	RedirectionURL string `json:"redirectionUrl" binding:"required"`
}

func (r *HomepageRequest) Validate() error {
	if strings.TrimSpace(r.UserCohort) == "" {
		return errors.New("userCohort is required")
	}

	if strings.TrimSpace(r.Location.City) == "" {
		return errors.New("location.city is required")
	}

	if r.Location.Latitude < -90 || r.Location.Latitude > 90 {
		return errors.New("location.latitude must be between -90 and 90")
	}

	if r.Location.Longitude < -180 || r.Location.Longitude > 180 {
		return errors.New("location.longitude must be between -180 and 180")
	}

	for i, banner := range r.Banners {
		if strings.TrimSpace(banner.Title) == "" {
			return errors.New("banner title is required at index " + string(rune(i)))
		}
		if strings.TrimSpace(banner.ImageURL) == "" {
			return errors.New("banner imageUrl is required at index " + string(rune(i)))
		}
	}

	for i, category := range r.Categories {
		if strings.TrimSpace(category.Name) == "" {
			return errors.New("category name is required at index " + string(rune(i)))
		}
		if strings.TrimSpace(category.IconURL) == "" {
			return errors.New("category iconUrl is required at index " + string(rune(i)))
		}
	}

	return nil
}