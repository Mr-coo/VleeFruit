package handler

import (
	"net/http"
	"strconv"

	"github.com/Mr-coo/VleeFruit/backend/internal/service"
	"github.com/gin-gonic/gin"
)

const defaultLimit = 50

// QueryHandler serves read-only listings of detections, images and readings.
type QueryHandler struct {
	detections *service.DetectionService
	readings   *service.ReadingService
}

func NewQueryHandler(detections *service.DetectionService, readings *service.ReadingService) *QueryHandler {
	return &QueryHandler{detections: detections, readings: readings}
}

func (h *QueryHandler) ListDetections(c *gin.Context) {
	results, err := h.detections.ListResults(c.Request.Context(), limitParam(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"detections": results})
}

func (h *QueryHandler) ListImages(c *gin.Context) {
	images, err := h.detections.ListImages(c.Request.Context(), limitParam(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": images})
}

func (h *QueryHandler) ListReadings(c *gin.Context) {
	readings, err := h.readings.List(c.Request.Context(), limitParam(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"readings": readings})
}

// limitParam reads ?limit=N, defaulting when absent or invalid.
func limitParam(c *gin.Context) int {
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultLimit
}
