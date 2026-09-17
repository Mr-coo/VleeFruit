package repository

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"gorm.io/gorm"
)

type ImageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// Create persists an image and (if attached) its detection result in one tx.
func (r *ImageRepository) Create(ctx context.Context, img *domain.Image) error {
	return r.db.WithContext(ctx).Create(img).Error
}

func (r *ImageRepository) List(ctx context.Context, limit int) ([]domain.Image, error) {
	var images []domain.Image
	q := r.db.WithContext(ctx).Preload("Detection").Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&images).Error
	return images, err
}
