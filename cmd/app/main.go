package main

import (
	"fmt"
	"log"

	"app_backend/internal/auth"
	"app_backend/internal/config"
	"app_backend/internal/db"
	httpServer "app_backend/internal/http"
	"context"
	"app_backend/internal/redis"
	// "github.com/redis/go-redis/v9"
	"app_backend/internal/http/handlers"
	"app_backend/internal/http/middleware"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/service"
	"app_backend/internal/sms"
	"app_backend/internal/socket"
	"app_backend/internal/worker"


	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {

	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	} else {
		fmt.Println(".env file loaded successfully")
	}

	// Load config
	cfg := config.Load()

	// Mongo
	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongo connect:", err)

	}
	rdb := redis.NewRedis()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("❌ Redis connection failed:", err)
	}
	log.Println("✅ Redis connected successfully")
	var database *mongo.Database = client.Database(cfg.DBName)
	fmt.Println("Mongo Connected → DB:", cfg.DBName)

	/* ---------------- SOCKET ---------------- */
	hub := socket.NewHub()
	emitter := socket.NewEmitter(hub)

	/* ---------------- REPOSITORIES ---------------- */
	paymentRepo := repository.NewPaymentRepository(database)
	userRepo := repository.NewUserRepo(database)
	providerRepo := repository.NewProviderRepo(database)
	otpRepo := repository.NewOTPRepo(database)
	locationRepo := repository.NewLocationRepo(database)
	homepageRepo := repository.NewHomepageRepo(database)
	acceptedServiceRepo := repository.NewAcceptedServiceRepo(database)
	complaintRepo := repository.NewComplaintRepo(database)

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
	refundWorker := worker.NewRefundWorker(
	rdb,
	paymentSvc, // 👈 PaymentService implements RefundProcessor
	)
	refundWorker.Start()

	var smsClient ports.SMSClient = sms.SmsTrigger()
	var tokenSvc ports.TokenService = auth.NewJWT(cfg.JWTSecret)

	otpQueue := worker.NewOTPQueue(smsClient)
	otpQueue.Start()
	defer otpQueue.Stop()

	userSvc := service.NewUserService(userRepo, otpRepo, tokenSvc, otpQueue)
	providerSvc := service.NewProviderService(providerRepo, otpRepo, tokenSvc, otpQueue, acceptedServiceRepo)
	locationSvc := service.NewLocationService(locationRepo)
	complaintSvc := service.NewComplaintService(complaintRepo, userRepo, providerRepo)
	homepageSvc := service.NewHomepageService(homepageRepo)

	/* ---------------- HANDLERS ---------------- */
	userHandler := handlers.NewUserHandler(userSvc)
	providerHandler := handlers.NewProviderHandler(providerSvc)
	locationHandler := handlers.NewLocationHandler(locationSvc)
	complaintHandler := handlers.NewComplaintHandler(complaintSvc)
	homepageHandler := handlers.NewHomepageHandler(homepageSvc)
	paymentHandler := handlers.NewPaymentHandler(paymentSvc)

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
	)

	log.Println("🚀 Server running on port:", cfg.HTTPPort)

	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error:", err)
	}
}
