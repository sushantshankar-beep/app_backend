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
	locationHandler *handlers.LocationHandler, // ✅ Add this line
) *gin.Engine {

	r := gin.Default()

	// USER ROUTES
	user := r.Group("/user")
	{
		// Public
		user.POST("/send-otp", userHandler.SendOTP)
		user.POST("/verify-otp", userHandler.VerifyOTP)

		// Auth
		user.GET("/profile", userAuth, userHandler.Profile)

		// Location
		user.POST("/location", userAuth, locationHandler.SaveUserLocation)
		user.GET("/location", userAuth, locationHandler.GetUserLocation)
	}

	// PROVIDER ROUTES
	provider := r.Group("/provider")
	{
		// Public
		provider.POST("/send-otp", providerHandler.SendOTP)
		provider.POST("/verify-otp", providerHandler.VerifyOTP)

		// Auth
		provider.GET("/profile", providerAuth, providerHandler.Profile)
		provider.POST("/location", providerAuth, locationHandler.SaveProviderLocation)
		provider.GET("/location", providerAuth, locationHandler.GetProviderLocation)
		provider.PUT("/profile-update", providerAuth, providerHandler.CreateOrUpdateProfile)
	}

	return r
}
