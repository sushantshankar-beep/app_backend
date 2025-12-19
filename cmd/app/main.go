package main

import (
	"context"
	"fmt"
	"log"

	"app_backend/internal/auth"
	"app_backend/internal/config"
	"app_backend/internal/db"
	httpServer "app_backend/internal/http"
	"app_backend/internal/http/handlers"
	"app_backend/internal/http/middleware"
	"app_backend/internal/ports"
	"app_backend/internal/redis"
	"app_backend/internal/repository"
	"app_backend/internal/service"
	"app_backend/internal/sms"
	"app_backend/internal/socket"
	"app_backend/internal/worker"

	"github.com/joho/godotenv"
	// "go.mongodb.org/mongo-driver/mongo"
)

func main() {

	/* ---------------- ENV ---------------- */
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env not found, using system env")
	} else {
		fmt.Println("✅ .env loaded")
	}

	cfg := config.Load()

	/* ---------------- MONGO ---------------- */
	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("❌ Mongo connect:", err)
	}
	db := client.Database(cfg.DBName)
	log.Println("✅ Mongo connected:", cfg.DBName)

	/* ---------------- REDIS ---------------- */
	rdb := redis.NewRedis()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("❌ Redis connection failed:", err)
	}
	log.Println("✅ Redis connected")

	/* ---------------- SOCKET ---------------- */
	hub := socket.NewHub()
	emitter := socket.NewEmitter(hub)

	/* ---------------- REPOSITORIES ---------------- */
	paymentRepo := repository.NewPaymentRepository(db)
	userRepo := repository.NewUserRepo(db)
	providerRepo := repository.NewProviderRepo(db)
	otpRepo := repository.NewOTPRepo(db)
	locationRepo := repository.NewLocationRepo(db)
	homepageRepo := repository.NewHomepageRepo(db)
	acceptedServiceRepo := repository.NewAcceptedServiceRepo(db)
	complaintRepo := repository.NewComplaintRepo(db)
	amcRepo := repository.NewAMCRepo(db)
	cancellationRepo := repository.NewCancellationRepo(db)
	serviceCatalogRepo := repository.NewServiceCatalogRepo(db)

	/* ---------------- SERVICES ---------------- */
	notificationSvc := service.NewFirebaseNotificationService()

	paymentSvc := service.NewPaymentService(
		paymentRepo,
		emitter,
		acceptedServiceRepo,
		providerRepo,
		notificationSvc,
		cfg.PayUKey,
		cfg.PayUSalt,
		cfg.PayUBaseURL,
		cfg.BaseURL,
		rdb,
	)

	// 🔁 Refund async worker
	refundWorker := worker.NewRefundWorker(rdb, paymentSvc)
	refundWorker.Start()

	// 🔐 Auth + OTP
	var smsClient ports.SMSClient = sms.SmsTrigger()
	var tokenSvc ports.TokenService = auth.NewJWT(cfg.JWTSecret)

	otpQueue := worker.NewOTPQueue(smsClient)
	otpQueue.Start()
	defer otpQueue.Stop()

	userSvc := service.NewUserService(userRepo, otpRepo, tokenSvc, otpQueue)
	providerSvc := service.NewProviderService(
		providerRepo,
		otpRepo,
		tokenSvc,
		otpQueue,
		acceptedServiceRepo,
	)
	locationSvc := service.NewLocationService(locationRepo)
	complaintSvc := service.NewComplaintService(complaintRepo, userRepo, providerRepo)
	homepageSvc := service.NewHomepageService(homepageRepo)
	bookingSvc := service.NewBookingService(acceptedServiceRepo,userRepo,providerRepo,serviceCatalogRepo)

	// ✅ AMC validation
	amcValidationSvc := service.NewAMCValidationService(amcRepo)

	// ✅ Bidding service (FIXED)
	biddingSvc := service.NewBiddingService(
		rdb,
		emitter,
		acceptedServiceRepo,
		cancellationRepo,
	)
	serviceTrackingSvc := service.NewServiceTrackingService(
		acceptedServiceRepo,
		userRepo,
		providerRepo,
		emitter,
	)


	/* ---------------- HANDLERS ---------------- */
	userHandler := handlers.NewUserHandler(userSvc)
	providerHandler := handlers.NewProviderHandler(providerSvc)
	locationHandler := handlers.NewLocationHandler(locationSvc)
	complaintHandler := handlers.NewComplaintHandler(complaintSvc)
	homepageHandler := handlers.NewHomepageHandler(homepageSvc)
	paymentHandler := handlers.NewPaymentHandler(paymentSvc)
	amcValidationHandler := handlers.NewAMCValidationHandler(amcValidationSvc)
	biddingHandler := handlers.NewBiddingHandler(biddingSvc)
	bookingHandler := handlers.NewBookingHandler(bookingSvc)
	serviceTrackingHandler := handlers.NewServiceTrackingHandler(
		serviceTrackingSvc,
	)


	/* ---------------- MIDDLEWARE ---------------- */
	userAuth := middleware.AuthUser(tokenSvc)
	providerAuth := middleware.AuthProvider(tokenSvc)
	

	/* ---------------- ROUTER ---------------- */
	r := httpServer.SetupRouter(
		userHandler,
		providerHandler,
		userAuth,
		providerAuth,
		locationHandler,
		complaintHandler,
		homepageHandler,
		paymentHandler,
		biddingHandler,
		amcValidationHandler,
		hub,
		bookingHandler,
		serviceTrackingHandler,
	)
	log.Println("🚀 Server running on port:", cfg.HTTPPort)

	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("❌ Server error:", err)
	}
}
