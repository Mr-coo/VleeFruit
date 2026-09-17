package domain

import "time"

// Ripeness is the classification produced by the detector.
type Ripeness string

const (
	RipenessUnripe   Ripeness = "unripe"
	RipenessRipe     Ripeness = "ripe"
	RipenessOverripe Ripeness = "overripe"
	RipenessSpoiled  Ripeness = "spoiled"
)

// DetectionResult is the ripeness verdict for a single image.
type DetectionResult struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ImageID      uint      `gorm:"index;not null" json:"image_id"`
	Ripeness     Ripeness  `json:"ripeness"`
	Confidence   float64   `json:"confidence"`
	ModelVersion string    `json:"model_version"`
	CreatedAt    time.Time `json:"created_at"`
}
