package inference

import (
	"context"
	"errors"
)

// ErrModelNotConfigured is returned when no model file is available, so the
// server runs but inference is disabled.
var ErrModelNotConfigured = errors.New("no model configured (set MODEL_PATH to an existing .onnx file)")

// Config configures the inference stack.
type Config struct {
	// ModelPath is the path to the .onnx model file. Empty/missing disables inference.
	ModelPath string
	// SharedLibPath is the path to libonnxruntime.so / onnxruntime.dll.
	SharedLibPath string
	// InputName / OutputName are the model's tensor names (YOLO defaults shown).
	InputName  string
	OutputName string
	// InputSize is the square input dimension (e.g. 640).
	InputSize int
	// Labels maps class IDs (in training order) to human labels.
	Labels []string
	// ConfThreshold / IoUThreshold tune decoding and NMS.
	ConfThreshold float32
	IoUThreshold  float32
}

// Output is the raw model output: a flat float32 buffer plus its shape.
type Output struct {
	Data  []float32
	Shape []int64
}

// Runner runs a model on an already-preprocessed NCHW tensor and returns the
// raw output. Implementations are backed by the ONNX Runtime (cgo).
type Runner interface {
	// Run executes the model on input (len == 3*size*size, NCHW float32).
	Run(ctx context.Context, input []float32, size int) (*Output, error)
	Close() error
}

// disabledRunner is used when no model is available. It lets the server start
// and serve every other route; inference calls return a clear error.
type disabledRunner struct{ err error }

func (d disabledRunner) Run(context.Context, []float32, int) (*Output, error) {
	return nil, d.err
}

func (d disabledRunner) Close() error { return nil }
