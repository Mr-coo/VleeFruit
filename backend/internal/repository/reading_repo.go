package repository

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"gorm.io/gorm"
)

type ReadingRepository struct {
	db *gorm.DB
}

func NewReadingRepository(db *gorm.DB) *ReadingRepository {
	return &ReadingRepository{db: db}
}

func (r *ReadingRepository) Create(ctx context.Context, reading *domain.SensorReading) error {
	return r.db.WithContext(ctx).Create(reading).Error
}

func (r *ReadingRepository) List(ctx context.Context, limit int) ([]domain.SensorReading, error) {
	var readings []domain.SensorReading
	q := r.db.WithContext(ctx).Order("recorded_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&readings).Error
	return readings, err
}
