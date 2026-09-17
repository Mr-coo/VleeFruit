package domain

import "time"

// Image is a stored capture from a device. The bytes live in object storage
// (ObjectKey/Bucket); this row is the metadata plus its detection result.
type Image struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	BatchID     *uint            `json:"batch_id,omitempty"`
	DeviceID    string           `gorm:"index" json:"device_id"`
	ObjectKey   string           `gorm:"not null" json:"object_key"`
	Bucket      string           `json:"bucket"`
	ContentType string           `json:"content_type"`
	SizeBytes   int64            `json:"size_bytes"`
	Width       int              `json:"width"`
	Height      int              `json:"height"`
	CapturedAt  time.Time        `json:"captured_at"`
	CreatedAt   time.Time        `json:"created_at"`
	Detection   *DetectionResult `gorm:"foreignKey:ImageID" json:"detection,omitempty"`
}
