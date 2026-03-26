package handlers

import (
	"net/http"
    "strings"
	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/s3"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "fmt"
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

	if len(req.Phone) != 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone must be 10 digits"})
		return
	}

	if req.Phone[0] < '6' || req.Phone[0] > '9' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone must start with 6-9"})
		return
	}
	
	for _, ch := range req.Phone {
		if ch < '0' || ch > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone must contain only digits"})
			return
		}
	}

	if err := h.svc.SendOTP(c, req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
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

	if len(req.Phone) != 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone must be 10 digits"})
		return
	}

	if req.Phone[0] < '6' || req.Phone[0] > '9' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone must start with 6-9"})
		return
	}

	for _, ch := range req.Phone {
		if ch < '0' || ch > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone must contain only digits"})
			return
		}
	}

	resp, err := h.svc.VerifyOTP(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) Profile(c *gin.Context) {
	userObjIDAny, exists := c.Get(middleware.ContextKeyUserObjectID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userObjID := userObjIDAny.(primitive.ObjectID)
	user, err := h.svc.GetProfile(c.Request.Context(), userObjID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (h *UserHandler) CreateOrUpdateUserProfile(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req map[string]any
	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		req = make(map[string]any)
		fields := []string{"name", "email", "fcmToken", "selectedCity", "appStateStatus"}
		for _, field := range fields {
			if val := c.PostForm(field); val != "" {
				req[field] = val
			}
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
	}

	if urls, exists := s3.GetUploadedURLs(c, "profileImage"); exists && len(urls) > 0 {
		req["imageUrl"] = urls[0]
	}

	user, action, err := h.svc.CreateOrUpdateProfile(c.Request.Context(),domain.UserID(userID),req,)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": action, "user": user})
}

func (h *UserHandler) Logout(c *gin.Context) {
	userObjID := c.MustGet(middleware.ContextKeyUserObjectID).(primitive.ObjectID)
	userID := domain.UserID(userObjID.Hex())

	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	
	if err := h.svc.Logout(c.Request.Context(), userID, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userObjIDAny, exists := c.Get(middleware.ContextKeyUserObjectID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userObjID := userObjIDAny.(primitive.ObjectID)

	if err := h.svc.DeleteUser(c.Request.Context(), userObjID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}
