
package http

import (
	"app_backend/internal/http/handlers"
	// "app_backend/internal/repository"
	"github.com/gin-gonic/gin"
	// "github.com/redis/go-redis/v9"
	"app_backend/internal/s3"
	"app_backend/internal/socket"
)

func SetupRouter(userHandler *handlers.UserHandler,providerHandler *handlers.ProviderHandler,userAuth gin.HandlerFunc,providerAuth gin.HandlerFunc,locationHandler *handlers.LocationHandler,complaintHandler *handlers.ComplaintHandler,homepageHandler *handlers.HomepageHandler,paymentHandler *handlers.PaymentHandler,biddingHandler *handlers.BiddingHandler,amcValidationHandler *handlers.AMCValidationHandler,hub *socket.Hub,bookingHandler *handlers.BookingHandler,serviceTrackingHandler *handlers.ServiceTrackingHandler,kycHandler *handlers.KYCHandler,invoiceHandler *handlers.InvoiceHandler,userVehicleHandler *handlers.UserVehicleHandler,providerStatus *handlers.ProviderStatusHandler,metaHandler *handlers.MetaHandler,s3Uploader *s3.Uploader,imageUploadS3Handler *handlers.ImageUploadS3Handler) *gin.Engine {

	r := gin.Default()

	// === Payment Routes ===
	payment := r.Group("/payment")
	{
		payment.POST("/initiate",userAuth, paymentHandler.InitiatePayment)
		payment.POST("/webhook", paymentHandler.PayUWebhook)
		payment.POST("/refund", userAuth, paymentHandler.Refund)
		payment.GET("/verify/:serviceId", userAuth, paymentHandler.VerifyPayment)
	}
	// === User Routes ===
	user := r.Group("/user")
	{
		user.POST("/send-otp", userHandler.SendOTP)
		user.POST("/verify-otp", userHandler.VerifyOTP)
		user.GET("/vehicleNumber/:vehicleNumber",userAuth, userVehicleHandler.GetVehicleByNumber)
		user.POST("/vehicle", userAuth,userVehicleHandler.SaveVehicle)
		user.GET("/vehicleData", userAuth, userVehicleHandler.GetVehicleData)
		user.GET("/profile", userAuth, userHandler.Profile)
		user.PUT("/profile", userAuth, s3Uploader.Upload([]s3.FieldConfig{
			{FormFieldName: "profileImage",ContextKey:    "profileImage",}}), 
		userHandler.CreateOrUpdateUserProfile)
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
		bid.POST("/reject", biddingHandler.RejectBid)
	}
	// === Websocket handling ===

	r.GET("/ws", socket.HandleWebSocket(hub))
	booking := r.Group("/booking")
	{
		booking.GET("/details/:serviceId",userAuth,bookingHandler.GetBookingDetails)
	}
	invoice := r.Group("/invoice", userAuth)
	{
		invoice.GET("/:serviceId", invoiceHandler.GetInvoice)
		invoice.GET("/:serviceId/download", invoiceHandler.DownloadInvoice)
	}
	// ===Provider kyc =====
	kyc := r.Group("/provider/kyc", providerAuth)
	{
		kyc.POST("", s3Uploader.Upload([]s3.FieldConfig{
			{FormFieldName: "aadhaarFront", ContextKey: "aadhaarFront"},
			{FormFieldName: "aadhaarBack", ContextKey: "aadhaarBack"},
			{FormFieldName: "pan", ContextKey: "pan"},
		}), kycHandler.CreateOrUpdateKYC)
		kyc.GET("", kycHandler.GetKYC)
	}

	// === Provider Routes ===
	provider := r.Group("/provider")
	{
		provider.POST("/send-otp", providerHandler.SendOTP)
		provider.POST("/verify-otp", providerHandler.VerifyOTP)
		provider.GET("/profile", providerAuth, providerHandler.Profile)
		provider.PUT("/profile-update", providerAuth, s3Uploader.Upload([]s3.FieldConfig{
		    {
			  FormFieldName: "profileImage",
			  ContextKey:    "profileImage",
		    },
	    }), providerHandler.CreateOrUpdateProfile )
		provider.POST("/location", providerAuth, locationHandler.SaveProviderLocation)
		provider.GET("/location", providerAuth, locationHandler.GetProviderLocation)
		provider.PUT("/profile", providerAuth, providerHandler.CreateOrUpdateProfile)
		provider.PUT("/dashboard", providerAuth, providerHandler.Dashboard)
		provider.POST("/online", providerAuth, providerStatus.GoOnline)
		provider.POST("/offline", providerAuth, providerStatus.GoOffline)
		provider.GET("/my-services", providerAuth, providerHandler.GetMyAllServices)
		provider.GET("/my-service/:id", providerAuth, providerHandler.GetMyService)
		provider.POST("/raise-complaint", providerAuth, complaintHandler.RaiseComplaint)
		provider.GET("/complaints", providerAuth, complaintHandler.GetProviderComplaints)
		provider.POST("/bid", providerAuth,biddingHandler.PlaceBid)
	}
	meta := r.Group("/meta")
	{
		meta.GET("/brands", metaHandler.GetBrands)
		meta.GET("/services", metaHandler.GetServices)
	}

	imageUpload := r.Group("/upload-image")
	{
		imageUpload.POST("", s3Uploader.Upload([]s3.FieldConfig{
				{
					FormFieldName: "image",
					ContextKey:    "image",
				},
		}), imageUploadS3Handler.UploadSingle)
	}
	
	if homepageHandler != nil {
		r.GET("/homepage", homepageHandler.GetHomepage)
	}

	return r
}
