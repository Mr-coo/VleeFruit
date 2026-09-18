// Package detection analyses a fruit image across three tasks: ripeness
// classification, defect detection, and fruit-size estimation. The real
// implementation runs an ONNX model in-process; until a model is added, a
// noop stub keeps the app building and booting.
package detection

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
)

// Result is the detector's verdict for one image across all three tasks.
type Result struct {
	Ripeness           domain.Ripeness
	RipenessConfidence float64

	Defective        bool
	DefectConfidence float64

	Size     float64
	SizeUnit string

	ModelVersion string
}

// Detector turns image bytes into a Result (ripeness, defect and size).
type Detector interface {
	Detect(ctx context.Context, img []byte, contentType string) (Result, error)
}
