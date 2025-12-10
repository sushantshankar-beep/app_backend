package middleware

import (
	"net/http"
	"strings"

	"app_backend/internal/ports"

	"github.com/gin-gonic/gin"
)

const ContextKeyUserID = "userID"

func AuthUser(tokenSvc ports.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		id, typ, err := tokenSvc.Parse(raw)
		if err != nil || typ != "user" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user token"})
			return
		}

		c.Set(ContextKeyUserID, id)
		c.Next()
	}
}

func AuthProvider(tokenSvc ports.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		id, typ, err := tokenSvc.Parse(raw)
		if err != nil || typ != "provider" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid provider token"})
			return
		}

		c.Set(ContextKeyUserID, id)
		c.Next()
	}
}