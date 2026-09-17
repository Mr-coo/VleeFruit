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
	// Cheap deterministic pick so results vary across images without a model.
	levels := []domain.Ripeness{
		domain.RipenessUnripe,
		domain.RipenessRipe,
		domain.RipenessOverripe,
		domain.RipenessSpoiled,
	}
	idx := len(img) % len(levels)
	return Result{
		Ripeness:     levels[idx],
		Confidence:   0,
		ModelVersion: "noop",
	}, nil
}
