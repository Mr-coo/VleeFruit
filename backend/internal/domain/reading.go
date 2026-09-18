package domain

import "time"

// Metric is the kind of value a sensor reading carries.
type Metric string

const (
	MetricEthylene    Metric = "ethylene"
	MetricTemperature Metric = "temperature"
	MetricHumidity    Metric = "humidity"
)

// Valid reports whether m is one of the known metrics.
func (m Metric) Valid() bool {
	switch m {
	case MetricEthylene, MetricTemperature, MetricHumidity:
		return true
	default:
		return false
	}
}

// SensorReading is a single telemetry value published by a device over MQTT.
type SensorReading struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceID   string    `gorm:"index" json:"device_id"`
	BatchID    *uint     `json:"batch_id,omitempty"`
	Metric     Metric    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}
