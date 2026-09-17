package database

import (
	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"gorm.io/gorm"
)

// AutoMigrate creates/updates the tables for all domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Device{},
		&domain.Batch{},
		&domain.Image{},
		&domain.DetectionResult{},
		&domain.SensorReading{},
		&domain.ServiceAccount{},
	)
}
