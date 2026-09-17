package domain

import "time"

// ServiceAccount is a non-device MQTT identity (e.g. the backend itself).
// Devices authenticate via the devices table; service accounts live here so
// Postgres stays the single source of truth for broker auth.
type ServiceAccount struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Superuser    bool      `gorm:"default:false" json:"superuser"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName keeps the broker's auth queries explicit and stable.
func (ServiceAccount) TableName() string { return "mqtt_service_accounts" }
