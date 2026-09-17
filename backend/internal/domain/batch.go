package domain

import "time"

// Batch is a lot of fruit being monitored for ripeness.
type Batch struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	FruitType       string    `json:"fruit_type"`
	BatchCode       string    `gorm:"uniqueIndex" json:"batch_code"`
	StorageLocation string    `json:"storage_location"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
