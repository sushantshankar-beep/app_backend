package handlers

import (
	"net/http"

	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	svc *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) GetBookingDetails(c *gin.Context) {
	serviceID := c.Param("serviceId")

	resp, err := h.svc.BuildBookingScreen(
		c.Request.Context(),
		serviceID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *BookingHandler) GetUserBookings(c *gin.Context) {
	userID := c.Param("userID")
	status := c.Query("status")

	if userID == "" || status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID and status are required"})
		return
	}

	bookings, err := h.svc.GetUserBookings(c.Request.Context(), userID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}
