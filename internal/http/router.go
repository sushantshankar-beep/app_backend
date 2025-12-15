package http

import (
	"app_backend/internal/http/handlers"

	"github.com/gin-gonic/gin"
	"app_backend/internal/payment"
)

func SetupRouter(
	userHandler *handlers.UserHandler,
	providerHandler *handlers.ProviderHandler,
	userAuth gin.HandlerFunc,
	providerAuth gin.HandlerFunc,
	locationHandler *handlers.LocationHandler,
	complaintHandler *handlers.ComplaintHandler,
	homepageHandler *handlers.HomepageHandler,
) *gin.Engine {

	r := gin.Default()
	paySvc := payment.NewService(transactionRepo)
	payHandler := payment.NewHandler(paySvc)
	r.POST("/payment/initiate", payHandler.Initiate)
	r.POST("/payment/verify", payHandler.Verify)
	r.POST("/api/payment/webhook/success", payHandler.Webhook)
	r.POST("/api/payment/webhook/failure", payHandler.Webhook)

	user := r.Group("/user")
	{
		user.POST("/send-otp", userHandler.SendOTP)
		user.POST("/verify-otp", userHandler.VerifyOTP)
		user.GET("/profile", userAuth, userHandler.Profile)
		user.POST("/location", userAuth, locationHandler.SaveUserLocation)
		user.GET("/location", userAuth, locationHandler.GetUserLocation)
		user.POST("/raise-complaint", userAuth, complaintHandler.RaiseComplaint)
		user.GET("/complaints", userAuth, complaintHandler.GetMyComplaints)
	}

	provider := r.Group("/provider")
	{
		provider.POST("/send-otp", providerHandler.SendOTP)
		provider.POST("/verify-otp", providerHandler.VerifyOTP)
		provider.GET("/profile", providerAuth, providerHandler.Profile)
		provider.PUT("/profile-update", providerAuth, providerHandler.CreateOrUpdateProfile)
		provider.POST("/location", providerAuth, locationHandler.SaveProviderLocation)
		provider.GET("/location", providerAuth, locationHandler.GetProviderLocation)
		provider.PUT("/profile", providerAuth, providerHandler.CreateOrUpdateProfile)
		provider.PUT("/dashboard", providerAuth, providerHandler.Dashboard)
		provider.GET("/my-services", providerAuth, providerHandler.GetMyAllServices)
		provider.GET("/my-service/:id", providerAuth, providerHandler.GetMyService)
		provider.POST("/raise-complaint", providerAuth, complaintHandler.RaiseComplaint)
		provider.GET("/complaints", providerAuth, complaintHandler.GetProviderComplaints)
	}

	return r
}
