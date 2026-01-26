package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"
)

type ComplaintHandler struct {
	svc *service.ComplaintService
}

func NewComplaintHandler(s *service.ComplaintService) *ComplaintHandler {
	return &ComplaintHandler{svc: s}
}

func (h *ComplaintHandler) RaiseComplaint(c *gin.Context) {

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	raisedBy, ok := req["raisedBy"].(string)
	if !ok || (raisedBy != "user" && raisedBy != "provider") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raisedBy must be user or provider"})
		return
	}

	var authenticatedID string
	if raisedBy == "user" {
		authenticatedID = c.GetString(middleware.ContextKeyUserID)
	} else {
		authenticatedID = c.GetString(middleware.ContextKeyProviderID)
	}

	if authenticatedID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	complaint, err := h.svc.RaiseComplaint(
		c.Request.Context(),
		req,
		raisedBy,
		authenticatedID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Complaint saved successfully",
		"data":    complaint,
	})
}

func (h *ComplaintHandler) GetMyComplaints(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	list, err := h.svc.GetUserComplaints(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *ComplaintHandler) GetProviderComplaints(c *gin.Context) {
	providerID := c.GetString(middleware.ContextKeyUserID)

	list, err := h.svc.GetProviderComplaints(c, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, list)
}