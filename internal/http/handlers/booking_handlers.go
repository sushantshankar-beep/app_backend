package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
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
