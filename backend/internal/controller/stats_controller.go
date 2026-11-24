package controller

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Counter reports how many classifications were recorded per label. It is
// satisfied by *store.Store; a nil Counter means stats storage is unavailable.
type Counter interface {
	Counts(ctx context.Context) (map[string]int, error)
}

// StatsController serves classification counts for the dashboard.
type StatsController struct {
	counter Counter
}

// NewStatsController builds a StatsController. The counter may be nil, in which
// case the endpoint responds 503.
func NewStatsController(counter Counter) *StatsController {
	return &StatsController{counter: counter}
}

// Stats responds with per-label classification counts and their total.
// @Summary      Classification counts
// @Description  Returns how many Gemini classifications were recorded per label, plus the total.
// @Tags         stats
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /api/v1/stats [get]
func (s *StatsController) Stats(c *gin.Context) {
	if s.counter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stats storage unavailable"})
		return
	}

	counts, err := s.counter.Counts(c.Request.Context())
	if err != nil {
		log.Printf("stats: counts failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stats query failed"})
		return
	}

	total := 0
	for _, n := range counts {
		total += n
	}

	c.JSON(http.StatusOK, gin.H{"counts": counts, "total": total})
}
