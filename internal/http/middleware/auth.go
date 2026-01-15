package middleware

import (
	"net/http"
	"strings"

	"app_backend/internal/ports"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"fmt"
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
func AuthUser(tokenSvc ports.TokenService) gin.HandlerFunc {
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
		fmt.Println(userIDStr)
		userObjID, err := primitive.ObjectIDFromHex(userIDStr)
		fmt.Println(userObjID,err)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user id",
			})
			return
		}

		c.Set(ContextKeyUserID, userIDStr)
		c.Set(ContextKeyUserObjectID, userObjID)

		c.Next()
	}
}


func AuthProvider(tokenSvc ports.TokenService) gin.HandlerFunc {
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
				"error": "invalid provider token",
			})
			return
		}

		providerObjID, err := primitive.ObjectIDFromHex(providerIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid provider id",
			})
			return
		}

		c.Set(ContextKeyProviderID, providerIDStr)
		c.Set(ContextKeyProviderObjID, providerObjID)

		c.Next()
	}
}
