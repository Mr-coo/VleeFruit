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
// @Summary      Health check
// @Description  Returns the service liveness status.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /healthz [get]
func (h *HealthController) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "vleefruit-backend",
	})
}
