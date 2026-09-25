package yolo

import (
	"fmt"
	"sort"
)

// Box is an axis-aligned bounding box in ORIGINAL image pixel coordinates.
type Box struct {
	X1, Y1, X2, Y2 float32
}

func (b Box) Width() float32  { return b.X2 - b.X1 }
func (b Box) Height() float32 { return b.Y2 - b.Y1 }
func (b Box) area() float32 {
	w, h := b.Width(), b.Height()
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// Detection is one decoded object: its class, confidence, and box.
type Detection struct {
	ClassID    int
	Label      string
	Confidence float32
	Box        Box
}

// Decode turns a raw YOLOv8 detection output into detections in original-image
// coordinates, applying a confidence threshold and per-class NMS.
//
// It expects the Ultralytics ONNX export layout: shape [1, 4+nc, na], where the
// first 4 channels are box center/size (cx, cy, w, h) in the letterboxed input
// space, followed by nc per-class scores (already sigmoid-activated). There is
// no separate objectness channel (YOLOv8/v11 convention).
func Decode(data []float32, shape []int64, pp PreprocessResult, origW, origH int, labels []string, confThres, iouThres float32) ([]Detection, error) {
	if len(shape) != 3 || shape[0] != 1 {
		return nil, fmt.Errorf("unexpected output shape %v (want [1, 4+nc, na])", shape)
	}
	channels := int(shape[1])
	na := int(shape[2])
	if channels < 5 {
		return nil, fmt.Errorf("output has %d channels, need at least 5 (4 box + >=1 class)", channels)
	}
	if len(data) < channels*na {
		return nil, fmt.Errorf("output data len %d < channels*na %d", len(data), channels*na)
	}
	nc := channels - 4

	// val(c, a) accesses the channel-major flat buffer.
	val := func(c, a int) float32 { return data[c*na+a] }

	var cands []Detection
	for a := 0; a < na; a++ {
		bestClass, bestScore := 0, float32(0)
		for k := 0; k < nc; k++ {
			s := val(4+k, a)
			if s > bestScore {
				bestScore, bestClass = s, k
			}
		}
		if bestScore < confThres {
			continue
		}

		cx, cy, w, h := val(0, a), val(1, a), val(2, a), val(3, a)
		box := mapToOriginal(cx, cy, w, h, pp, origW, origH)
		if box.area() <= 0 {
			continue
		}
		cands = append(cands, Detection{
			ClassID:    bestClass,
			Label:      labelFor(labels, bestClass),
			Confidence: bestScore,
			Box:        box,
		})
	}

	return nms(cands, iouThres), nil
}

// mapToOriginal converts a box from letterboxed input space (center form) to
// original-image pixel coordinates (corner form), reversing the letterbox.
func mapToOriginal(cx, cy, w, h float32, pp PreprocessResult, origW, origH int) Box {
	scale := float32(pp.Scale)
	x1 := (cx - w/2 - float32(pp.PadX)) / scale
	y1 := (cy - h/2 - float32(pp.PadY)) / scale
	x2 := (cx + w/2 - float32(pp.PadX)) / scale
	y2 := (cy + h/2 - float32(pp.PadY)) / scale
	return Box{
		X1: clamp(x1, 0, float32(origW)),
		Y1: clamp(y1, 0, float32(origH)),
		X2: clamp(x2, 0, float32(origW)),
		Y2: clamp(y2, 0, float32(origH)),
	}
}

// nms applies greedy per-class non-maximum suppression.
func nms(dets []Detection, iouThres float32) []Detection {
	sort.Slice(dets, func(i, j int) bool { return dets[i].Confidence > dets[j].Confidence })

	kept := make([]Detection, 0, len(dets))
	suppressed := make([]bool, len(dets))
	for i := range dets {
		if suppressed[i] {
			continue
		}
		kept = append(kept, dets[i])
		for j := i + 1; j < len(dets); j++ {
			if suppressed[j] || dets[j].ClassID != dets[i].ClassID {
				continue
			}
			if iou(dets[i].Box, dets[j].Box) > iouThres {
				suppressed[j] = true
			}
		}
	}
	return kept
}

func iou(a, b Box) float32 {
	ix1 := maxf(a.X1, b.X1)
	iy1 := maxf(a.Y1, b.Y1)
	ix2 := minf(a.X2, b.X2)
	iy2 := minf(a.Y2, b.Y2)
	iw, ih := ix2-ix1, iy2-iy1
	if iw <= 0 || ih <= 0 {
		return 0
	}
	inter := iw * ih
	union := a.area() + b.area() - inter
	if union <= 0 {
		return 0
	}
	return inter / union
}

func labelFor(labels []string, id int) string {
	if id >= 0 && id < len(labels) {
		return labels[id]
	}
	return fmt.Sprintf("class_%d", id)
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
