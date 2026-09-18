package handler

import (
	"net/http"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/service"
	"github.com/Mr-coo/VleeFruit/backend/internal/validation"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	svc *service.DeviceService
}

func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

type provisionRequest struct {
	DeviceID string            `json:"device_id" binding:"required"`
	Name     string            `json:"name"`
	Location string            `json:"location"`
	Kind     domain.DeviceKind `json:"kind"`
}

// Provision registers a device and returns its API key ONCE.
func (h *DeviceHandler) Provision(c *gin.Context) {
	var req provisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.DeviceID(req.DeviceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !req.Kind.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be one of: camera, gas, temperature, humidity"})
		return
	}

	device, rawKey, err := h.svc.Provision(c.Request.Context(), req.DeviceID, req.Name, req.Location, req.Kind)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"device":  device,
		"api_key": rawKey, // shown once; store it on the device now
	})
}

// List returns all devices (without key material).
func (h *DeviceHandler) List(c *gin.Context) {
	devices, err := h.svc.List(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// Revoke disables a device.
func (h *DeviceHandler) Revoke(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if err := validation.DeviceID(deviceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), deviceID); err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "revoked", "device_id": deviceID})
}

// RotateKey issues a fresh API key for a device and returns it once.
func (h *DeviceHandler) RotateKey(c *gin.Context) {
	deviceID := c.Param("deviceID")
	if err := validation.DeviceID(deviceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rawKey, err := h.svc.RotateKey(c.Request.Context(), deviceID)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": deviceID, "api_key": rawKey})
}
