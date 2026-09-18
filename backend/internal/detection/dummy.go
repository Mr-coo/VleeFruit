package detection

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
)

// dummyDetector always returns the same static verdict, regardless of the
// image. It exists purely for testing the pipeline until a real model is wired
// in — every call yields identical, predictable output.
type dummyDetector struct{}

// NewDummy returns a Detector that always produces the same static result.
func NewDummy() Detector { return dummyDetector{} }

func (dummyDetector) Detect(_ context.Context, _ []byte, _ string) (Result, error) {
	return Result{
		Ripeness:           domain.RipenessRipe,
		RipenessConfidence: 0.92,
		Defective:          false,
		DefectConfidence:   0.97,
		Size:               75.0,
		SizeUnit:           "mm",
		ModelVersion:       "dummy-static-v1",
	}, nil
}
