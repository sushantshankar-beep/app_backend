package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"app_backend/internal/config"
	"app_backend/internal/db"
	"app_backend/internal/redis"
	"app_backend/internal/repository"
	"app_backend/internal/service"
	"app_backend/internal/socket"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system env")
	} else {
		fmt.Println(".env loaded")
	}

	cfg := config.Load()

	ctx := context.Background()

	client, err := db.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("Mongo connect:", err)
	}

	dbConn := client.Database(cfg.DBName)

	log.Println("✅ Mongo connected:", cfg.DBName)

	rdb := redis.NewRedis()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis ping failed:", err)
	}

	log.Println("✅ Redis connected")

	hub := socket.NewHub()
	emitter := socket.NewEmitter(hub)

	acceptedServiceRepo := repository.NewAcceptedServiceRepo(dbConn)
	// providerRepo := repository.NewProviderRepo(dbConn)

	timeoutCron := service.NewProviderAssignmentTimeoutCron(
		rdb,
		emitter,
		acceptedServiceRepo,
	)

	go timeoutCron.Run(ctx)

	log.Println("⏰ Provider-assignment timeout cron started")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig

	log.Println("🛑 Cron stopped")
}
