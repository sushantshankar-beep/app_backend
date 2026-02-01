package handlers

import (
	"app_backend/internal/service"
	
	"net/http"

	"github.com/gin-gonic/gin"
	"fmt"
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

func (h *BookingHandler) GetUserBookingDetail(c *gin.Context) {
	userID := c.Param("userID")
	bookingID := c.Param("serviceID")

	booking, err := h.svc.GetUserBookingDetails( c.Request.Context(), userID, bookingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"booking": booking})
}

func (h *BookingHandler) GetProviderBookings(c *gin.Context) {
	providerID := c.Param("providerID")
	status := c.Query("status")

	if providerID == "" || status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID and status are required"})
		return
	}

	bookings, err := h.svc.GetProviderBookings(c.Request.Context(), providerID, status)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bookings)
}

func (h *BookingHandler) GetProviderBookingDetail(c *gin.Context) {
	providerID := c.Param("providerID")
	bookingID := c.Param("serviceID")
	fmt.Println("this is provider id",providerID)
	fmt.Println("this is booking id",bookingID)

	booking, err := h.svc.GetProviderBookingDetails( c.Request.Context(), providerID, bookingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"booking": booking})
}

func (h *BookingHandler) GetUserExpenses(c *gin.Context) {
	userID := c.Param("userID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID is required"})
		return
	}

	expenses, totalExpense, err := h.svc.GetUserExpenses(c.Request.Context(), userID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totalExpense": totalExpense,
		"expenses":     expenses,
	})
}

func (h *BookingHandler) GetProviderDashboard(c *gin.Context) {
	providerID := c.Param("providerID")

	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	dashboard, err := h.svc.GetProviderDashboard(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}


func (h *BookingHandler) GetProviderEarnings(c *gin.Context) {
	providerID := c.Param("providerID")

	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	earnings, err := h.svc.GetProviderEarnings(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, earnings)
}

func (h *BookingHandler) GetProviderTodayEarnings(c *gin.Context) {
	providerID := c.Param("providerID")

	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	earnings, err := h.svc.GetProviderTodayEarnings(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, earnings)
}

func (h *BookingHandler) GetProviderSettledEarnings(c *gin.Context) {
	providerID := c.Param("providerID")

	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	resp, err := h.svc.GetProviderSettledEarnings(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
