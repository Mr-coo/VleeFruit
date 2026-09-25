package inference

import (
	"context"
	"errors"
	"image"
)

// ErrModelNotConfigured is returned when no model path was provided, so the
// server runs but inference is disabled.
var ErrModelNotConfigured = errors.New("no model configured (set MODEL_PATH)")

// Config configures the inference runner.
type Config struct {
	// ModelPath is the path to the .onnx model file. Empty disables inference.
	ModelPath string
	// SharedLibPath is the path to libonnxruntime.so / onnxruntime.dll.
	SharedLibPath string
	// InputName / OutputName are the model's tensor names (YOLO defaults shown).
	InputName  string
	OutputName string
	// InputSize is the square input dimension (e.g. 640).
	InputSize int
}

// Output is the raw model output: a flat float32 buffer plus its shape.
// Decoding it (e.g. YOLO box decode + NMS) is model-specific and done by the
// caller/service layer.
type Output struct {
	Data  []float32
	Shape []int64
}

// Runner runs a model over a decoded image. Implementations are selected at
// build time via the "onnx" tag.
type Runner interface {
	Infer(ctx context.Context, img image.Image) (*Output, error)
	Close() error
}

// disabledRunner is used when inference is unavailable or no model is set. It
// lets the server start and serve every other route; the inference endpoint
// returns a clear error.
type disabledRunner struct{ err error }

func (d disabledRunner) Infer(context.Context, image.Image) (*Output, error) {
	return nil, d.err
}

func (d disabledRunner) Close() error { return nil }
