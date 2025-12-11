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

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	} else {
		fmt.Println(".env file loaded successfully")
	}

	cfg := config.Load()

	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongo connect:", err)
	}

	var database *mongo.Database = client.Database(cfg.DBName)

	fmt.Println("Mongo Connected → DB:", cfg.DBName)

	userRepo := repository.NewUserRepo(database)
	providerRepo := repository.NewProviderRepo(database)
	otpRepo := repository.NewOTPRepo(database)
	locationRepo := repository.NewLocationRepo(database)
	homepageRepo := repository.NewHomepageRepo(database)
	acceptedServiceRepo := repository.NewAcceptedServiceRepo(database)
	complaintRepo := repository.NewComplaintRepo(database)
	var smsClient ports.SMSClient = sms.NewDummySMS()
	var tokenSvc ports.TokenService = auth.NewJWT(cfg.JWTSecret)

	otpQueue := worker.NewOTPQueue(smsClient)
	otpQueue.Start()
	defer otpQueue.Stop()

	userSvc := service.NewUserService(userRepo, otpRepo, tokenSvc, otpQueue)
	providerSvc := service.NewProviderService(providerRepo, otpRepo, tokenSvc, otpQueue, acceptedServiceRepo)
	locationSvc := service.NewLocationService(locationRepo)
	complaintSvc := service.NewComplaintService(complaintRepo, userRepo, providerRepo)
	homepageSvc := service.NewHomepageService(homepageRepo)
	userHandler := handlers.NewUserHandler(userSvc)
	providerHandler := handlers.NewProviderHandler(providerSvc)
	locationHandler := handlers.NewLocationHandler(locationSvc)
	complaintRepoHandler := handlers.NewComplaintHandler(complaintSvc)
	homepageHandler := handlers.NewHomepageHandler(homepageSvc)
	userAuth := middleware.AuthUser(tokenSvc)
	providerAuth := middleware.AuthProvider(tokenSvc)

	r := httpServer.SetupRouter(
		userHandler,
		providerHandler,
		userAuth,
		providerAuth,
		locationHandler,
		complaintRepoHandler,
		homepageHandler,
	)

	log.Println("Server running on port:", cfg.HTTPPort)

	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("server error:", err)
	}
}
