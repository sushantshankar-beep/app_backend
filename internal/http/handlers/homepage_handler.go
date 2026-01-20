package handlers

import (
	"net/http"

	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
)

type HomepageHandler struct {
	svc *service.HomepageService
}

func NewHomepageHandler(svc *service.HomepageService) *HomepageHandler {
	return &HomepageHandler{svc: svc}
}

func (h *HomepageHandler) GetHomepage(c *gin.Context) {

	country := c.GetHeader("X-Country")
	state := c.GetHeader("X-State")
	city := c.GetHeader("X-City")

	homepage, err := h.svc.GetHomepage(c.Request.Context(),country,state,city)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "homepage not found"})
		return
	}
	c.JSON(http.StatusOK, homepage)
}
