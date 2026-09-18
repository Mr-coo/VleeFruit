package domain

import "time"

// DeviceKind is the type of IoT device reporting to the system.
type DeviceKind string

const (
	DeviceKindCamera      DeviceKind = "camera"
	DeviceKindGas         DeviceKind = "gas"
	DeviceKindTemperature DeviceKind = "temperature"
	DeviceKindHumidity    DeviceKind = "humidity"
)

// Valid reports whether k is one of the known device kinds.
func (k DeviceKind) Valid() bool {
	switch k {
	case DeviceKindCamera, DeviceKindGas, DeviceKindTemperature, DeviceKindHumidity:
		return true
	default:
		return false
	}
}

// DeviceStatus controls whether a device may authenticate.
type DeviceStatus string

const (
	DeviceStatusActive  DeviceStatus = "active"
	DeviceStatusRevoked DeviceStatus = "revoked"
)

// Device is a provisioned IoT unit. DeviceID is the public identifier used as
// the MQTT username; KeyHash is the bcrypt hash of its per-device API key (the
// raw key is shown once at provisioning and never stored).
type Device struct {
	ID         uint         `gorm:"primaryKey" json:"id"`
	DeviceID   string       `gorm:"uniqueIndex;not null" json:"device_id"`
	Name       string       `json:"name"`
	Location   string       `json:"location"`
	Kind       DeviceKind   `json:"kind"`
	KeyHash    string       `gorm:"not null" json:"-"`
	Status     DeviceStatus `gorm:"default:active;index" json:"status"`
	LastSeenAt *time.Time   `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}
