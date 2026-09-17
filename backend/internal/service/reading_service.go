package service

import (
	"context"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
)

// ReadingService persists sensor telemetry arriving from devices.
type ReadingService struct {
	repo *repository.ReadingRepository
}

func NewReadingService(repo *repository.ReadingRepository) *ReadingService {
	return &ReadingService{repo: repo}
}

// Record stores a single reading. RecordedAt defaults to now when zero.
func (s *ReadingService) Record(ctx context.Context, reading *domain.SensorReading) error {
	if reading.RecordedAt.IsZero() {
		reading.RecordedAt = time.Now()
	}
	return s.repo.Create(ctx, reading)
}

// List returns the most recent readings.
func (s *ReadingService) List(ctx context.Context, limit int) ([]domain.SensorReading, error) {
	return s.repo.List(ctx, limit)
}
