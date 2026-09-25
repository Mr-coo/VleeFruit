package inference

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// envOnce guards one-time initialization of the shared ONNX Runtime environment.
var envOnce sync.Once
var envErr error

func initEnv(sharedLibPath string) error {
	envOnce.Do(func() {
		if sharedLibPath != "" {
			ort.SetSharedLibraryPath(sharedLibPath)
		}
		if !ort.IsInitialized() {
			envErr = ort.InitializeEnvironment()
		}
	})
	return envErr
}

type onnxRunner struct {
	cfg     Config
	session *ort.DynamicAdvancedSession
	mu      sync.Mutex // ORT sessions are not guaranteed re-entrant; serialize Run.
}

// NewRunner loads the model into a reusable session. A missing/empty ModelPath
// yields a disabled runner so the server still starts.
func NewRunner(cfg Config) (Runner, error) {
	if cfg.ModelPath == "" {
		log.Printf("inference: no MODEL_PATH set; inference disabled")
		return disabledRunner{err: ErrModelNotConfigured}, nil
	}
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		// Model path set but file missing: start disabled rather than crashing.
		log.Printf("inference: model %q not found; inference disabled until it exists", cfg.ModelPath)
		return disabledRunner{err: ErrModelNotConfigured}, nil
	}
	if cfg.InputName == "" {
		cfg.InputName = "images"
	}
	if cfg.OutputName == "" {
		cfg.OutputName = "output0"
	}

	if err := initEnv(cfg.SharedLibPath); err != nil {
		return nil, fmt.Errorf("init onnxruntime env: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(
		cfg.ModelPath,
		[]string{cfg.InputName},
		[]string{cfg.OutputName},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("load model %q: %w", cfg.ModelPath, err)
	}

	log.Printf("inference: loaded model %q", cfg.ModelPath)
	return &onnxRunner{cfg: cfg, session: session}, nil
}

func (r *onnxRunner) Run(_ context.Context, input []float32, size int) (*Output, error) {
	if len(input) != 3*size*size {
		return nil, fmt.Errorf("input len %d != 3*%d*%d", len(input), size, size)
	}

	inputShape := ort.NewShape(1, 3, int64(size), int64(size))
	inputTensor, err := ort.NewTensor(inputShape, input)
	if err != nil {
		return nil, fmt.Errorf("create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	outputs := []ort.Value{nil} // nil => auto-allocated by Run
	r.mu.Lock()
	err = r.session.Run([]ort.Value{inputTensor}, outputs)
	r.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("run inference: %w", err)
	}

	outTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type %T (expected float32 tensor)", outputs[0])
	}
	defer outTensor.Destroy()

	shape := outTensor.GetShape()
	src := outTensor.GetData()
	// Copy out of the ORT-owned buffer before it is destroyed.
	data := make([]float32, len(src))
	copy(data, src)
	dims := make([]int64, len(shape))
	copy(dims, shape)

	return &Output{Data: data, Shape: dims}, nil
}

func (r *onnxRunner) Close() error {
	if r.session != nil {
		return r.session.Destroy()
	}
	return nil
}
