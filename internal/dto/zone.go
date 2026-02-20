package dto

type CheckZoneRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

type CheckZoneResponse struct {
	Serviceable bool   `json:"serviceable"`
	ZoneName    string `json:"zoneName,omitempty"`
}
