package detection

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
)

// noopDetector is a placeholder used until a real ONNX model is wired in.
// It returns a deterministic, clearly-fake verdict so the end-to-end flow
// (ingest -> store -> detect -> persist -> publish) can be exercised.
type noopDetector struct{}

// NewNoop returns a Detector that does no real inference.
func NewNoop() Detector { return noopDetector{} }

func (noopDetector) Detect(_ context.Context, img []byte, _ string) (Result, error) {
	// Cheap deterministic picks so results vary across images without a model.
	levels := []domain.Ripeness{
		domain.RipenessUnripe,
		domain.RipenessRipe,
		domain.RipenessOverripe,
		domain.RipenessSpoiled,
	}
	n := len(img)
	return Result{
		Ripeness:           levels[n%len(levels)],
		RipenessConfidence: 0,
		Defective:          n%2 == 0,
		DefectConfidence:   0,
		Size:               float64(50 + n%50), // fake diameter in mm
		SizeUnit:           "mm",
		ModelVersion:       "noop",
	}, nil
}
