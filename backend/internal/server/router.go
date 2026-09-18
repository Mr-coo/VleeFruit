// Package server builds the Gin HTTP engine and runs it with graceful shutdown.
package server

import (
	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/Mr-coo/VleeFruit/backend/internal/handler"
	"github.com/Mr-coo/VleeFruit/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Handlers bundles the REST handlers the router wires up.
type Handlers struct {
	Health *handler.HealthHandler
	Device *handler.DeviceHandler
	Query  *handler.QueryHandler
}

// NewRouter builds the Gin engine. /healthz is public; everything under
// /api/v1 sits behind the admin bearer token.
func NewRouter(cfg *config.Config, h Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.RequestID())

	r.GET("/healthz", h.Health.Check)

	admin := r.Group("/api/v1",
		middleware.BodyLimit(cfg.HTTP.MaxBodyBytes),
		middleware.AdminAuth(cfg.AdminToken))
	{
		admin.POST("/devices", h.Device.Provision)
		admin.GET("/devices", h.Device.List)
		admin.POST("/devices/:deviceID/revoke", h.Device.Revoke)
		admin.POST("/devices/:deviceID/rotate-key", h.Device.RotateKey)

		admin.GET("/detections", h.Query.ListDetections)
		admin.GET("/images", h.Query.ListImages)
		admin.GET("/readings", h.Query.ListReadings)
	}

	return r
}
