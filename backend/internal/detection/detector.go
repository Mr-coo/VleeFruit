// Package detection classifies fruit ripeness from an image. The real
// implementation runs an ONNX model in-process; until a model is added, a
// noop stub keeps the app building and booting.
package detection

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
)

// Result is the detector's verdict for one image.
type Result struct {
	Ripeness     domain.Ripeness
	Confidence   float64
	ModelVersion string
}

// Detector turns image bytes into a ripeness Result.
type Detector interface {
	Detect(ctx context.Context, img []byte, contentType string) (Result, error)
}
