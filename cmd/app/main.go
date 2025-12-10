package main

import (
	"fmt"
	"log"

	"app_backend/internal/auth"
	"app_backend/internal/config"
	"app_backend/internal/db"
	httpServer "app_backend/internal/http"
	"app_backend/internal/http/handlers"
	"app_backend/internal/http/middleware"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/service"
	"app_backend/internal/sms"
	"app_backend/internal/worker"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {

	// ------------------------------
	// Load .env file
	// ------------------------------
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found — using system environment variables")
	} else {
		fmt.Println(".env file loaded successfully")
	}

	// ------------------------------
	// Load config
	// ------------------------------
	cfg := config.Load()

	// ------------------------------
	// Connect MongoDB
	// ------------------------------
	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongo connect:", err)
	}

	var database *mongo.Database = client.Database(cfg.DBName)

	fmt.Println("Mongo Connected → DB:", cfg.DBName)

	// ------------------------------
	// Initialize Repositories
	// ------------------------------
	userRepo := repository.NewUserRepo(database)
	providerRepo := repository.NewProviderRepo(database)
	otpRepo := repository.NewOTPRepo(database)
	locationRepo := repository.NewLocationRepo(database) // ⭐ NEW

	// ------------------------------
	// SMS + JWT
	// ------------------------------
	var smsClient ports.SMSClient = sms.NewDummySMS()
	var tokenSvc ports.TokenService = auth.NewJWT(cfg.JWTSecret)

	// ------------------------------
	// OTP Queue Worker
	// ------------------------------
	otpQueue := worker.NewOTPQueue(smsClient)
	otpQueue.Start()
	defer otpQueue.Stop()

	// ------------------------------
	// Services
	// ------------------------------
	userSvc := service.NewUserService(userRepo, otpRepo, tokenSvc, otpQueue)
	providerSvc := service.NewProviderService(providerRepo, otpRepo, tokenSvc, otpQueue)
	locationSvc := service.NewLocationService(locationRepo) // ⭐ NEW

	// ------------------------------
	// Handlers
	// ------------------------------
	userHandler := handlers.NewUserHandler(userSvc)
	providerHandler := handlers.NewProviderHandler(providerSvc)
	locationHandler := handlers.NewLocationHandler(locationSvc) // ⭐ NEW

	// ------------------------------
	// Middleware
	// ------------------------------
	userAuth := middleware.AuthUser(tokenSvc)
	providerAuth := middleware.AuthProvider(tokenSvc)

	// ------------------------------
	// Router
	// ------------------------------
	r := httpServer.SetupRouter(
		userHandler,
		providerHandler,
		userAuth,
		providerAuth,
		locationHandler, // ⭐ NEW
	)

	// ------------------------------
	// Start Server
	// ------------------------------
	log.Println("🚀 Server running on port:", cfg.HTTPPort)

	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error:", err)
	}
}
