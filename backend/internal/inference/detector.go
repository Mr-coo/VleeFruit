package inference

import (
	"context"
	"image"

	"github.com/Mr-coo/VleeFruit/backend/internal/inference/yolo"
)

// Detector is the high-level entry point: preprocess -> model -> YOLO decode.
type Detector struct {
	runner    Runner
	size      int
	labels    []string
	confThres float32
	iouThres  float32
}

// NewDetector wraps a Runner with decoding parameters from cfg, applying
// sensible defaults.
func NewDetector(runner Runner, cfg Config) *Detector {
	size := cfg.InputSize
	if size == 0 {
		size = 640
	}
	conf := cfg.ConfThreshold
	if conf <= 0 {
		conf = 0.25
	}
	iou := cfg.IoUThreshold
	if iou <= 0 {
		iou = 0.45
	}
	return &Detector{
		runner:    runner,
		size:      size,
		labels:    cfg.Labels,
		confThres: conf,
		iouThres:  iou,
	}
}

// Detect runs the full pipeline and returns detections in original-image pixels.
func (d *Detector) Detect(ctx context.Context, img image.Image) ([]yolo.Detection, error) {
	pp := yolo.Preprocess(img, d.size)

	out, err := d.runner.Run(ctx, pp.Data, d.size)
	if err != nil {
		return nil, err
	}

	b := img.Bounds()
	return yolo.Decode(out.Data, out.Shape, pp, b.Dx(), b.Dy(), d.labels, d.confThres, d.iouThres)
}

// Close releases the underlying model session.
func (d *Detector) Close() error { return d.runner.Close() }
