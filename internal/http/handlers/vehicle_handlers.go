package handlers

import (
	"net/http"
     "strings"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type MetaHandler struct {
	svc *service.MetaService
}

func NewMetaHandler(s *service.MetaService) *MetaHandler {
	return &MetaHandler{svc: s}
}

func (h *MetaHandler) GetBrands(c *gin.Context) {
	vehicle := strings.ToLower(c.Query("vehicle"))
	if vehicle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle required"})
		return
	}

	data, err := h.svc.GetBrands(c.Request.Context(), vehicle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"brands": data})
}

func (h *MetaHandler) GetServices(c *gin.Context) {
	vehicle := c.Query("vehicle")
	if vehicle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vehicle required"})
		return
	}

	data, err := h.svc.GetServices(c.Request.Context(), vehicle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"services": data})
}
