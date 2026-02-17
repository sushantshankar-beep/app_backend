package handlers

import (
	"app_backend/internal/http/middleware"
	"app_backend/internal/s3"
	"app_backend/internal/service"
	"fmt"
	"net/http"
	"strings"
   "strconv"
   "math"
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

	pageInt, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limitInt, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if pageInt <= 0 {
		pageInt = 1
	}
	if limitInt <= 0 {
		limitInt = 20
	}

	page := int64(pageInt)
	limit := int64(limitInt)

	list, total, err := h.svc.GetUserComplaints(c, userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data":       list,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
}

func (h *ComplaintHandler) GetProviderComplaints(c *gin.Context) {
	providerID := c.GetString(middleware.ContextKeyProviderID)

	pageInt, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limitInt, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if pageInt <= 0 {
		pageInt = 1
	}
	if limitInt <= 0 {
		limitInt = 20
	}

	page := int64(pageInt)
	limit := int64(limitInt)

	list, total, err := h.svc.GetProviderComplaints(c, providerID,page,limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data":       list,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
}

func (h *ComplaintHandler) GetUserComplaintByBooking(c *gin.Context) {
	acceptedServiceID := c.Param("acceptedServiceId")
	
	if acceptedServiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "acceptedServiceId is required",
		})
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized: user authentication required",
		})
		return
	}

	complaint, err := h.svc.GetUserComplaintByAcceptedService(c.Request.Context(), acceptedServiceID, userID)
	if err != nil {
		if err.Error() == "no complaint found for this booking" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err.Error() == "unauthorized: you can only view your own complaints" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": complaint,
	})
}

func (h *ComplaintHandler) GetProviderComplaintByBooking(c *gin.Context) {
	acceptedServiceID := c.Param("acceptedServiceId")
	
	if acceptedServiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "acceptedServiceId is required",
		})
		return
	}

	providerID := c.GetString(middleware.ContextKeyProviderID)
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized: provider authentication required",
		})
		return
	}

	complaint, err := h.svc.GetProviderComplaintByAcceptedService(c.Request.Context(), acceptedServiceID, providerID)
	if err != nil {
		if err.Error() == "no complaint found for this booking" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": complaint,
	})
}