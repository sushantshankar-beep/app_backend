package handlers

import (
	"app_backend/internal/http/middleware"
	"app_backend/internal/s3"
	"app_backend/internal/service"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ComplaintHandler struct {
	svc *service.ComplaintService
}

func NewComplaintHandler(s *service.ComplaintService) *ComplaintHandler {
	return &ComplaintHandler{svc: s}
}

func (h *ComplaintHandler) RaiseComplaint(c *gin.Context) {
	raisedBy := c.PostForm("raisedBy")
	if raisedBy != "user" && raisedBy != "provider" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "raisedBy is required and must be 'user' or 'provider'",
		})
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

	photoURLs, _ := s3.GetUploadedURLs(c, "complaint_photos")

	seq, err := h.svc.GetNextComplaintSequence(c.Request.Context())
	if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate complaint number"})
        return
    }
    complaintNumber := fmt.Sprintf("CMP%05d", seq)

	complaint, err := h.svc.RaiseComplaint(
		c.Request.Context(),
		c.PostForm("acceptedService"),
		c.PostForm("problem"),
		photoURLs,
		raisedBy,
		authenticatedID,
		complaintNumber, 
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Complaint saved successfully",
		"data":    complaint,
	})
}

func getString(req map[string]any, key string) (string, bool) {
	v, ok := req[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok && s != ""
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
	providerID := c.GetString(middleware.ContextKeyProviderID)
	list, err := h.svc.GetProviderComplaints(c, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, list)
}