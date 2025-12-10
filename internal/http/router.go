package http

import (
	"app_backend/internal/http/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	userHandler *handlers.UserHandler,
	providerHandler *handlers.ProviderHandler,
	userAuth gin.HandlerFunc,
	providerAuth gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()

	user := r.Group("/user")
	{
		user.POST("/send-otp", userHandler.SendOTP)
		user.POST("/verify-otp", userHandler.VerifyOTP)
		user.GET("/profile", userAuth, userHandler.Profile)
	}

	provider := r.Group("/provider")
	{
		provider.POST("/send-otp", providerHandler.SendOTP)
		provider.POST("/verify-otp", providerHandler.VerifyOTP)
		provider.GET("/profile", providerAuth, providerHandler.Profile)
		provider.PUT("/profile-update", providerAuth, providerHandler.CreateOrUpdateProfile)
	}

	return r
}
