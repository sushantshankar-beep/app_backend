package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"app_backend/internal/service"
)

type AMCValidationHandler struct {
	svc *service.AMCValidationService
}

func NewAMCValidationHandler(svc *service.AMCValidationService) *AMCValidationHandler {
	return &AMCValidationHandler{svc: svc}
}

func (h *AMCValidationHandler) ValidateProblems(c *gin.Context) {
	var req struct {
		VehicleNumber string   `json:"vehicleNumber"`
		Issues        []string `json:"selectedIssues"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	result, err := h.svc.ValidateIssues(
		c.Request.Context(),
		req.VehicleNumber,
		req.Issues,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
