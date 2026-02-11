package handlers

import (
	"app_backend/internal/service"
	"strconv"
	"net/http"
    "math"
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
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	bookings,total, err := h.svc.GetUserBookings(c.Request.Context(), userID, status, page, limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"bookings": bookings,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"totalPages":  totalPages,
	})
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	bookings,total, err := h.svc.GetProviderBookings(c.Request.Context(), providerID, status,page,limit)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	response := gin.H{
		"bookings": bookings.Bookings,
		"page":     page,
		"limit":    limit,
		"total":    total,
		"totalPages": totalPages,
	}

	c.JSON(http.StatusOK, response)
}

func (h *BookingHandler) GetProviderBookingDetail(c *gin.Context) {
	providerID := c.Param("providerID")
	bookingID := c.Param("serviceID")


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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	expenses, totalExpense, total, err := h.svc.GetUserExpenses(c.Request.Context(), userID, page, limit)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"totalExpense": totalExpense,
		"expenses":     expenses,
		"page":         page,
		"limit":        limit,
		"total":        total,
		"totalPages":   totalPages,
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
