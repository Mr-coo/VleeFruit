package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthController handles health/liveness checks.
type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// Health responds with the service status.
func (h *HealthController) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "vleefruit-backend",
	})
}
