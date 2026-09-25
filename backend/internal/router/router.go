package router

import (
	"github.com/Mr-coo/VleeFruit/backend/internal/controller"
	"github.com/gin-gonic/gin"
)

// New builds the Gin engine (the HTTP layer) and wires routes to controllers.
func New(c *controller.Controllers) *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", c.Health.Health)

	api := r.Group("/api/v1")
	{
		// Register feature routes here, e.g.:
		// api.POST("/analyze", c.Analysis.Analyze)
		_ = api
	}

	return r
}
