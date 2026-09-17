// Package repository holds Postgres data-access logic for the domain models.
package repository

import (
	"context"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, d *domain.Device) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DeviceRepository) List(ctx context.Context) ([]domain.Device, error) {
	var devices []domain.Device
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&devices).Error
	return devices, err
}

func (r *DeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*domain.Device, error) {
	var d domain.Device
	err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepository) SetStatus(ctx context.Context, deviceID string, status domain.DeviceStatus) error {
	return r.db.WithContext(ctx).Model(&domain.Device{}).
		Where("device_id = ?", deviceID).
		Update("status", status).Error
}

func (r *DeviceRepository) UpdateKeyHash(ctx context.Context, deviceID, keyHash string) error {
	return r.db.WithContext(ctx).Model(&domain.Device{}).
		Where("device_id = ?", deviceID).
		Update("key_hash", keyHash).Error
}

func (r *DeviceRepository) TouchLastSeen(ctx context.Context, deviceID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_seen_at", now).Error
}
