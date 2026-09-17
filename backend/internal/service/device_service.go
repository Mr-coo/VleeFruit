// Package service holds business logic that coordinates repositories,
// storage, detection and messaging.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type DeviceService struct {
	repo *repository.DeviceRepository
}

func NewDeviceService(repo *repository.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

// Provision creates a device, generates a high-entropy API key, stores only
// its bcrypt hash, and returns the raw key ONCE (it is never persisted).
func (s *DeviceService) Provision(ctx context.Context, deviceID, name, location string, kind domain.DeviceKind) (*domain.Device, string, error) {
	rawKey, err := generateKey()
	if err != nil {
		return nil, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash key: %w", err)
	}

	device := &domain.Device{
		DeviceID: deviceID,
		Name:     name,
		Location: location,
		Kind:     kind,
		KeyHash:  string(hash),
		Status:   domain.DeviceStatusActive,
	}
	if err := s.repo.Create(ctx, device); err != nil {
		return nil, "", fmt.Errorf("create device: %w", err)
	}
	return device, rawKey, nil
}

// List returns all devices.
func (s *DeviceService) List(ctx context.Context) ([]domain.Device, error) {
	return s.repo.List(ctx)
}

// Revoke marks a device as revoked; the broker rejects it on next connect.
func (s *DeviceService) Revoke(ctx context.Context, deviceID string) error {
	return s.repo.SetStatus(ctx, deviceID, domain.DeviceStatusRevoked)
}

// RotateKey issues a fresh API key for a device and returns it once.
func (s *DeviceService) RotateKey(ctx context.Context, deviceID string) (string, error) {
	rawKey, err := generateKey()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash key: %w", err)
	}
	if err := s.repo.UpdateKeyHash(ctx, deviceID, string(hash)); err != nil {
		return "", fmt.Errorf("update key: %w", err)
	}
	return rawKey, nil
}

// generateKey returns 32 bytes of randomness as a hex string.
func generateKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
