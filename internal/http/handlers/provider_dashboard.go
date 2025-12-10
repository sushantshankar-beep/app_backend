package handlers

import (
	"net/http"
	"strconv"

	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

func (h *ProviderHandler) Dashboard(c *gin.Context) {

	id := domain.ProviderID(c.GetString(middleware.ContextKeyUserID))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	status := c.DefaultQuery("status", "all")
	timeframe := c.Query("timeframe")

	result, err := h.svc.GetDashboardStats(c, id, page, limit, status, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Earnings data fetched successfully",
		"data":    result,
	})
}
