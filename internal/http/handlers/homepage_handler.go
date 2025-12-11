package handlers

import (
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/service"
	"app_backend/internal/validation"

	"github.com/gin-gonic/gin"
)

type HomepageHandler struct {
	svc *service.HomepageService
}

func NewHomepageHandler(s *service.HomepageService) *HomepageHandler {
	return &HomepageHandler{svc: s}
}

func (h *HomepageHandler) GetHomepage(c *gin.Context) {
	id := c.Param("id")

	homepage, err := h.svc.GetHomepage(c, id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Homepage not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Homepage fetched successfully",
		"data":    homepage,
	})
}

func (h *HomepageHandler) CreateOrUpdateHomepage(c *gin.Context) {
	var req validation.HomepageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	homepage, err := h.svc.CreateOrUpdateHomepage(c, req)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Homepage not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Homepage saved successfully",
		"data":    homepage,
	})
}