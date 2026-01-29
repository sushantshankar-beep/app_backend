package middleware

import (
	"net/http"
	"strings"

	"app_backend/internal/domain"
	"app_backend/internal/ports"

	"fmt"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
	===== CONTEXT KEYS =====
	NEVER reuse keys between user & provider
*/
const (
	ContextKeyUserID        = "userId"
	ContextKeyUserObjectID  = "userObjectId"

	ContextKeyProviderID    = "providerId"
	ContextKeyProviderObjID = "providerObjectId"
)

/*
	===== USER AUTH MIDDLEWARE =====
*/
func AuthUser(tokenSvc ports.TokenService,userRepo ports.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))

		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization must be Bearer token",
			})
			return
		}

		userIDStr, typ, err := tokenSvc.Parse(parts[1])
		if err != nil || typ != "user" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user token",
			})
			return
		}
		
		userObjID, err := primitive.ObjectIDFromHex(userIDStr)
		fmt.Println(userObjID,err)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user id",
			})
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), userObjID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid token or user not found",
			})
			return
		}

		if user.IsActive == domain.USER_BLACKLIST {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Account has been blacklisted",
			})
			return
		}

		if user.IsActive == domain.USER_DEACTIVE {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Account has been deactivated",
			})
			return
		}

		c.Set(ContextKeyUserID, userIDStr)
		c.Set(ContextKeyUserObjectID, userObjID)

		c.Next()
	}
}


func AuthProvider(tokenSvc ports.TokenService, providerRepo ports.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		token := strings.TrimPrefix(raw, "Bearer ")

		providerIDStr, typ, err := tokenSvc.Parse(token)
		if err != nil || typ != "provider" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired provider token",
			})
			return
		}

		
		providerObjID, err := primitive.ObjectIDFromHex(providerIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Invalid or expired token",
			})
			return
		}

		provider, err := providerRepo.FindByID(c.Request.Context(), domain.ProviderID(providerObjID.Hex()))
		if err != nil || provider == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid token or provider not found",
			})
			return
		}

	

		if provider.IsActive == domain.PROVIDER_BLACKLIST {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Account has been blacklisted",
			})
			return
		}

		if provider.IsActive == domain.PROVIDER_SUSPENDED {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Account has been suspended",
			})
			return
		}

		c.Set(ContextKeyProviderID, providerIDStr)
		c.Set(ContextKeyProviderObjID, providerObjID)

		c.Next()
	}
}
