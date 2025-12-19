package http

import (
	"app_backend/internal/http/handlers"
	// "app_backend/internal/repository"
	"github.com/gin-gonic/gin"
	// "github.com/redis/go-redis/v9"
	"app_backend/internal/socket"
)

func SetupRouter(
	userHandler *handlers.UserHandler,
	providerHandler *handlers.ProviderHandler,
	userAuth gin.HandlerFunc,
	providerAuth gin.HandlerFunc,
	locationHandler *handlers.LocationHandler,
	complaintHandler *handlers.ComplaintHandler,
	homepageHandler *handlers.HomepageHandler,
	paymentHandler *handlers.PaymentHandler,
	biddingHandler *handlers.BiddingHandler,
	amcValidationHandler *handlers.AMCValidationHandler,
	hub *socket.Hub,
	bookingHandler *handlers.BookingHandler,
	serviceTrackingHandler *handlers.ServiceTrackingHandler,
) *gin.Engine {

	r := gin.Default()

	// === Payment Routes ===
	payment := r.Group("/payment")
	{
		payment.POST("/initiate",userAuth, paymentHandler.InitiatePayment)
		payment.POST("/webhook", paymentHandler.PayUWebhook)
		payment.POST("/refund", userAuth, paymentHandler.Refund)
	}
	// === User Routes ===
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
	service := r.Group("/service")
	{
		service.POST("/validate-problems",userAuth,amcValidationHandler.ValidateProblems)
		service.GET("/:id/user-tracking", userAuth, serviceTrackingHandler.UserTracking)
		service.GET("/:id/provider-tracking", providerAuth, serviceTrackingHandler.ProviderTracking)
		service.POST("/:id/verify-otp", providerAuth, serviceTrackingHandler.VerifyOTP)
	}
	bid := r.Group("/bid", userAuth)
	{
		bid.POST("/find", biddingHandler.FindMechanics)
		bid.POST("/accept", biddingHandler.AcceptBid)
	}
	// === Websocket handling ===
	r.GET("/ws", socket.HandleWebSocket(hub))
	booking := r.Group("/booking")
	{
		booking.GET("/details/:serviceId",userAuth,bookingHandler.GetBookingDetails)
	}

	// === Provider Routes ===
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
		provider.POST("/bid", providerAuth,biddingHandler.PlaceBid)
	}
	if homepageHandler != nil {
		r.GET("/homepage", homepageHandler.GetHomepage)
	}

	return r
}
