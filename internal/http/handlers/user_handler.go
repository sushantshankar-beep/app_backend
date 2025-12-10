package handlers

import (
	"net/http"

	"app_backend/internal/http/middleware"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{svc: s}
}
func (h *UserHandler) SendOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}

	if err := h.svc.SendOTP(c, req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
}
func (h *UserHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, isNew, err := h.svc.VerifyOTP(c, req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"isNew": isNew,
	})
}
func (h *UserHandler) Profile(c *gin.Context) {
	id := c.GetString(middleware.ContextKeyUserID)
	c.JSON(http.StatusOK, gin.H{"id": id})
}
