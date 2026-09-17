package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// --- Postgres (GORM) ---
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=" + env("DB_HOST", "localhost") +
			" port=" + env("DB_PORT", "5432") +
			" user=" + env("DB_USER", "vleefruit") +
			" password=" + env("DB_PASSWORD", "vleefruit") +
			" dbname=" + env("DB_NAME", "vleefruit") +
			" sslmode=" + env("DB_SSLMODE", "disable")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	log.Println("connected to postgres")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("connected to redis")

	// --- HTTP server ---
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		status := gin.H{"status": "ok"}

		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			status["postgres"] = "down"
			c.JSON(http.StatusServiceUnavailable, status)
			return
		}
		status["postgres"] = "up"

		pingCtx, pingCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer pingCancel()
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			status["redis"] = "down"
			c.JSON(http.StatusServiceUnavailable, status)
			return
		}
		status["redis"] = "up"

		c.JSON(http.StatusOK, status)
	})

	addr := ":" + env("PORT", "8080")
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
