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

// DetectionResult is the per-image verdict across the three detection tasks:
// ripeness classification, defect detection, and fruit-size estimation.
type DetectionResult struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	ImageID uint `gorm:"index;not null" json:"image_id"`

	// Ripeness classification.
	Ripeness           Ripeness `json:"ripeness"`
	RipenessConfidence float64  `json:"ripeness_confidence"`

	// Defect detection.
	Defective        bool    `json:"defective"`
	DefectConfidence float64 `json:"defect_confidence"`

	// Fruit-size estimate (numeric value in SizeUnit, e.g. "mm" diameter).
	Size     float64 `json:"size"`
	SizeUnit string  `json:"size_unit"`

	ModelVersion string    `json:"model_version"`
	CreatedAt    time.Time `json:"created_at"`
}
