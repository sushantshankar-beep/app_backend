
package main

import (
	"context"
	"fmt"
	"log"

	"app_backend/internal/auth"
	"app_backend/internal/config"
	"app_backend/internal/db"
	"app_backend/internal/events"
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
	"app_backend/internal/s3"
	"os"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {

	//env
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system env")
	} else {
		fmt.Println(".env loaded")
	}

	cfg := config.Load()

	//aws-S3
	awsSession, err := config.InitAWSSession()
	if err != nil {
		log.Fatal("Failed to initialize AWS session:", err)
	}
	log.Println("AWS session initialized successfully")
	

	s3Uploader := s3.NewUploader(awsSession, os.Getenv("AWS_BUCKET_NAME"),os.Getenv("AWS_S3_FOLDER"))

	//mongo
	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("Mongo connect:", err)
	}
	db := client.Database(cfg.DBName)
	log.Println("✅ Mongo connected:", cfg.DBName)
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL // "nats://127.0.0.1:4222"
	}

	//redis
	rdb := redis.NewRedis()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Redis connection failed:", err)
	}
	log.Println("Redis connected")

	//socket
	hub := socket.NewHub()
	emitter := socket.NewEmitter(hub)
	firebaseClient, err := service.InitFirebaseClient()
	if err != nil {
		log.Fatal("firebase init:", err)
	}
	service.TestFCMSend(firebaseClient)


	//repository
	paymentRepo := repository.NewPaymentRepository(db)
	userRepo := repository.NewUserRepo(db)
	providerRepo := repository.NewProviderRepo(db)
	otpRepo := repository.NewOTPRepo(db)
	locationRepo := repository.NewLocationRepo(db)
	homepageRepo := repository.NewHomepageRepo(db)
	acceptedServiceRepo := repository.NewAcceptedServiceRepo(db)
	complaintRepo := repository.NewComplaintRepo(db)
	amcRepo := repository.NewAMCRepo(db)
	// cancellationRepo := repository.NewCancellationRepo(db)
	serviceCatalogRepo := repository.NewServiceCatalogRepo(db)
	kycRepo := repository.NewKYCRepo(db)
	bidRepo := repository.NewBidRepo(db)
	invoiceRepo := repository.NewInvoiceRepo(db)
	vehicleRepo := repository.NewVehicleRepo(db)
	counterRepo := repository.NewCounterRepo(db)
	vehicleBrandRepo := repository.NewVehicleBrandRepo(db)
	serviceMasterRepo := repository.NewServiceMasterRepo(db)
	carBrandModelRepo := repository.NewCarBrandModelRepo(db)
	bikeBrandModelRepo := repository.NewBikeBrandModelRepo(db)
	deviceTokenRepo := repository.NewDeviceTokenRepo(db)
	ratingRepo := repository.NewRatingRepository(db)
	providerAgreementRepo := repository.NewAgreementRepo(db)
	settlementRepo := repository.NewSettlementHistoryRepository(db)

	// userVehicleRepo := repository.NewUserVehicleRepo(db)
	//SERVICES
	notificationSvc := service.NewFirebaseNotificationService(
		firebaseClient,
		deviceTokenRepo,
	)

	kycService := service.NewKYCService(kycRepo,providerRepo)
	userVehicleService := service.NewUserVehicleService(
		carBrandModelRepo,
		bikeBrandModelRepo,
		vehicleRepo,
		userRepo,
	)
	bus := events.NewBus(natsURL)
	worker.NewPaymentConsumer(ports.AcceptedServiceRepository(acceptedServiceRepo))

	worker.NewProviderConsumer(ports.AcceptedServiceRepository(acceptedServiceRepo))
	paymentSvc := service.NewPaymentService(
		paymentRepo,
		invoiceRepo,
		emitter,
		ports.AcceptedServiceRepository(acceptedServiceRepo),
		userRepo,
		ports.ProviderRepo(providerRepo),
		ports.NotificationService(notificationSvc),
		bus,
		cfg.PayUKey,
		cfg.PayUSalt,
		cfg.PayUBaseURL,
		cfg.BaseURL,
		rdb,
	)
	//Refund async worker
	refundWorker := worker.NewRefundWorker(rdb, paymentSvc)
	refundWorker.Start()

	// Auth + OTP
	var smsClient ports.SMSClient = sms.SmsTrigger()
	var tokenSvc ports.TokenService = auth.NewJWT(cfg.JWTSecret)

	otpQueue := worker.NewOTPQueue(smsClient)
	otpQueue.Start()
	defer otpQueue.Stop()

	userSvc := service.NewUserService(userRepo, otpRepo, tokenSvc, otpQueue, counterRepo)
	providerSvc := service.NewProviderService(
		providerRepo,
		counterRepo,
		otpRepo,
		tokenSvc,
		otpQueue,
		acceptedServiceRepo,
	)
	invoiceSvc := service.NewInvoiceService(invoiceRepo,acceptedServiceRepo,userRepo,providerRepo,paymentRepo)
	locationSvc := service.NewLocationService(locationRepo)
	complaintSvc := service.NewComplaintService(complaintRepo, userRepo, providerRepo,acceptedServiceRepo)
	homepageSvc := service.NewHomepageService(homepageRepo,rdb)
	bookingSvc := service.NewBookingService(acceptedServiceRepo, userRepo, providerRepo, serviceCatalogRepo,paymentRepo,settlementRepo)
	metaSvc := service.NewMetaService(rdb, vehicleBrandRepo, serviceMasterRepo)
	// AMC validation
	amcValidationSvc := service.NewAMCValidationService(amcRepo)
	agreementSvc := service.NewAgreementService(providerAgreementRepo,providerRepo)


	// Bidding service
	biddingSvc := service.NewBiddingService(rdb, emitter, acceptedServiceRepo,userRepo,bidRepo,providerRepo, counterRepo,notificationSvc)
	watchdog := worker.NewSearchWatchdog(
		ports.AcceptedServiceRepository(acceptedServiceRepo),
		biddingSvc,
	)

	watchdog.Start()
	
	serviceTrackingSvc := service.NewServiceTrackingService(acceptedServiceRepo, userRepo, providerRepo, emitter,notificationSvc,rdb)
	imageUploadS3 := service.NewImageUploadS3Service()
	ratingService := service.NewRatingService(ratingRepo,userRepo,providerRepo,acceptedServiceRepo)
	//HANDLERS
	userHandler := handlers.NewUserHandler(userSvc)
	userVehicleHandler := handlers.NewUserVehicleHandler(userVehicleService)
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
	kycHandler := handlers.NewKYCHandler(kycService)
	invoiceHandler := handlers.NewInvoiceHandler(invoiceSvc)
	providerStatusHandler := handlers.NewProviderStatusHandler(rdb)
	metaHandler := handlers.NewMetaHandler(metaSvc)
    imageUploadS3Handler := handlers.NewUploadHandler(imageUploadS3)
	deviceHandler := handlers.NewDeviceHandler(deviceTokenRepo)
	ratingHandler := handlers.NewRatingHandler(ratingService)
	providerAgreementHandler := handlers.NewAgreementHandler(agreementSvc)

	//middleware
	userAuth := middleware.AuthUser(tokenSvc,userRepo)
	providerAuth := middleware.AuthProvider(tokenSvc,providerRepo)
	r := httpServer.SetupRouter(userHandler, providerHandler, userAuth, providerAuth, locationHandler, complaintHandler, homepageHandler, paymentHandler, biddingHandler, amcValidationHandler, hub, bookingHandler, serviceTrackingHandler, kycHandler, invoiceHandler, userVehicleHandler, providerStatusHandler, metaHandler,s3Uploader,imageUploadS3Handler,deviceHandler,ratingHandler,providerAgreementHandler)
	log.Println("Server running on port:", cfg.HTTPPort)

	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal("Server error:", err)
	}
}
