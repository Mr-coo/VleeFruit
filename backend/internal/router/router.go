package router

import (
	"github.com/Mr-coo/VleeFruit/backend/internal/controller"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// New builds the Gin engine (the HTTP layer) and wires routes to controllers.
func New(c *controller.Controllers) *gin.Engine {
	r := gin.Default()

	// Swagger UI at /swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/healthz", c.Health.Health)

	api := r.Group("/api/v1")
	{
		// Register feature routes here, e.g.:
		// api.POST("/analyze", c.Analysis.Analyze)
		_ = api
	}

	return r
}
