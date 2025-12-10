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
	locationHandler *handlers.LocationHandler,
) *gin.Engine {

	r := gin.Default()

	user := r.Group("/user")
	{
		user.POST("/send-otp", userHandler.SendOTP)
		user.POST("/verify-otp", userHandler.VerifyOTP)
		user.GET("/profile", userAuth, userHandler.Profile)
		user.POST("/location", userAuth, locationHandler.SaveUserLocation)
		user.GET("/location", userAuth, locationHandler.GetUserLocation)
	}

	provider := r.Group("/provider")
	{
		provider.POST("/send-otp", providerHandler.SendOTP)
		provider.POST("/verify-otp",  providerHandler.VerifyOTP)
		provider.GET("/profile", providerAuth, providerHandler.Profile)
		provider.POST("/location", providerAuth, locationHandler.SaveProviderLocation)
		provider.GET("/location", providerAuth, locationHandler.GetProviderLocation)
		provider.PUT("/profile", providerAuth, providerHandler.CreateOrUpdateProfile)
		provider.GET("/my-services", providerAuth, providerHandler.GetMyAllServices)
        provider.GET("/my-service/:id", providerAuth, providerHandler.GetMyService)
	}

	return r
}