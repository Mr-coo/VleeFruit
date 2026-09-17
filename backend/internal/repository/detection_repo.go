package repository

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"gorm.io/gorm"
)

type DetectionRepository struct {
	db *gorm.DB
}

func NewDetectionRepository(db *gorm.DB) *DetectionRepository {
	return &DetectionRepository{db: db}
}

func (r *DetectionRepository) List(ctx context.Context, limit int) ([]domain.DetectionResult, error) {
	var results []domain.DetectionResult
	q := r.db.WithContext(ctx).Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&results).Error
	return results, err
}
