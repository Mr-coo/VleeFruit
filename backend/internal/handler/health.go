// Package handler holds the REST (Gin) handlers for the management/query API.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
	store storage.Storage
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client, store storage.Storage) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb, store: store}
}

// Check reports the liveness of Postgres, Redis and object storage.
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := gin.H{"status": "ok"}
	healthy := true

	if sqlDB, err := h.db.DB(); err != nil || sqlDB.Ping() != nil {
		status["postgres"] = "down"
		healthy = false
	} else {
		status["postgres"] = "up"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = "down"
		healthy = false
	} else {
		status["redis"] = "up"
	}

	if err := h.store.Ping(ctx); err != nil {
		status["storage"] = "down"
		healthy = false
	} else {
		status["storage"] = "up"
	}

	if !healthy {
		status["status"] = "degraded"
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}
	c.JSON(http.StatusOK, status)
}
