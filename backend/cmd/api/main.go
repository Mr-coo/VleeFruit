// Command api is the VleeFruit backend entrypoint. It wires config,
// Postgres, Redis, object storage, the detector, MQTT ingestion and the HTTP
// server, then runs until interrupted.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/cache"
	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/Mr-coo/VleeFruit/backend/internal/database"
	"github.com/Mr-coo/VleeFruit/backend/internal/detection"
	"github.com/Mr-coo/VleeFruit/backend/internal/handler"
	appmqtt "github.com/Mr-coo/VleeFruit/backend/internal/mqtt"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
	"github.com/Mr-coo/VleeFruit/backend/internal/server"
	"github.com/Mr-coo/VleeFruit/backend/internal/service"
	"github.com/Mr-coo/VleeFruit/backend/internal/storage"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// --- Infrastructure ---
	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		return err
	}
	if err := database.AutoMigrate(db); err != nil {
		return err
	}
	// The backend's own MQTT identity is a Postgres-backed superuser account,
	// so the broker authenticates it from the same source of truth as devices.
	if err := database.SeedServiceAccount(db, cfg.MQTT.Username, cfg.MQTT.Password, true); err != nil {
		return err
	}
	log.Println("postgres: connected and migrated")

	rdb, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		return err
	}
	log.Println("redis: connected")

	store, err := storage.NewMinIO(cfg.S3)
	if err != nil {
		return err
	}
	bucketCtx, cancelBucket := context.WithTimeout(context.Background(), 10*time.Second)
	if err := store.EnsureBucket(bucketCtx); err != nil {
		cancelBucket()
		return err
	}
	cancelBucket()
	log.Printf("storage: bucket %q ready", store.Bucket())

	// --- Repositories ---
	deviceRepo := repository.NewDeviceRepository(db)
	imageRepo := repository.NewImageRepository(db)
	detectionRepo := repository.NewDetectionRepository(db)
	readingRepo := repository.NewReadingRepository(db)

	// --- Services ---
	detector := detection.NewNoop() // swap for ONNX once a model is added
	deviceSvc := service.NewDeviceService(deviceRepo)
	ingestionSvc := service.NewIngestionService(store, detector, imageRepo)
	readingSvc := service.NewReadingService(readingRepo)
	detectionSvc := service.NewDetectionService(detectionRepo, imageRepo)

	// --- MQTT ingestion ---
	broker := appmqtt.NewBroker(ingestionSvc, readingSvc, deviceRepo)
	if err := broker.Connect(cfg.MQTT); err != nil {
		return err
	}
	defer broker.Stop()

	// --- HTTP ---
	handlers := server.Handlers{
		Health: handler.NewHealthHandler(db, rdb, store),
		Device: handler.NewDeviceHandler(deviceSvc),
		Query:  handler.NewQueryHandler(detectionSvc, readingSvc),
	}
	engine := server.NewRouter(cfg, handlers)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return server.Run(ctx, ":"+cfg.HTTPPort, engine)
}
