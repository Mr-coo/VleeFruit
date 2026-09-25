package yolo

import "testing"

// buildOutput lays out anchors into the YOLOv8 channel-major format
// [1, 4+nc, na]: data[c*na + a].
func buildOutput(nc int, anchors [][]float32) ([]float32, []int64) {
	na := len(anchors)
	channels := 4 + nc
	data := make([]float32, channels*na)
	for a, vals := range anchors {
		for c := 0; c < channels; c++ {
			data[c*na+a] = vals[c]
		}
	}
	return data, []int64{1, int64(channels), int64(na)}
}

// identityPP is a no-op letterbox (scale 1, no pad) for a size x size image.
func identityPP(size int) PreprocessResult {
	return PreprocessResult{Scale: 1, PadX: 0, PadY: 0, Size: size}
}

func TestDecodeBasicAndThreshold(t *testing.T) {
	labels := []string{"unripe", "ripe", "overripe"}
	// Two anchors: one strong "ripe", one below threshold.
	// vals = [cx, cy, w, h, s0, s1, s2]
	anchors := [][]float32{
		{100, 100, 40, 40, 0.1, 0.9, 0.05}, // ripe, conf 0.9
		{300, 300, 20, 20, 0.1, 0.1, 0.10}, // max 0.1 < 0.25 -> dropped
	}
	data, shape := buildOutput(3, anchors)

	dets, err := Decode(data, shape, identityPP(640), 640, 640, labels, 0.25, 0.45)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(dets) != 1 {
		t.Fatalf("want 1 detection, got %d", len(dets))
	}
	d := dets[0]
	if d.Label != "ripe" || d.ClassID != 1 {
		t.Errorf("want ripe/class1, got %s/class%d", d.Label, d.ClassID)
	}
	// box: cx=100,w=40 -> x1=80,x2=120 ; cy=100,h=40 -> y1=80,y2=120
	if d.Box.X1 != 80 || d.Box.X2 != 120 || d.Box.Y1 != 80 || d.Box.Y2 != 120 {
		t.Errorf("unexpected box %+v", d.Box)
	}
	if d.Box.Width() != 40 || d.Box.Height() != 40 {
		t.Errorf("want 40x40, got %vx%v", d.Box.Width(), d.Box.Height())
	}
}

func TestDecodeNMSSuppressesOverlap(t *testing.T) {
	labels := []string{"unripe", "ripe", "overripe"}
	// Two heavily-overlapping "ripe" boxes; NMS should keep the higher one.
	anchors := [][]float32{
		{100, 100, 40, 40, 0.0, 0.90, 0.0},
		{102, 102, 40, 40, 0.0, 0.80, 0.0}, // ~same box, lower conf
		{400, 400, 40, 40, 0.0, 0.70, 0.0}, // far away, kept
	}
	data, shape := buildOutput(3, anchors)

	dets, err := Decode(data, shape, identityPP(640), 640, 640, labels, 0.25, 0.45)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(dets) != 2 {
		t.Fatalf("want 2 after NMS, got %d", len(dets))
	}
	if dets[0].Confidence != 0.90 {
		t.Errorf("want top conf 0.90 kept, got %v", dets[0].Confidence)
	}
}

func TestDecodeLetterboxMapping(t *testing.T) {
	// Image was 320x160, letterboxed into 640: scale=2, pad y=160, pad x=0.
	pp := PreprocessResult{Scale: 2, PadX: 0, PadY: 160, Size: 640}
	// A box at input-space center (200,320) size (80,80):
	// x -> (200 - 0)/2 = 100 ; y -> (320 - 160)/2 = 80
	anchors := [][]float32{{200, 320, 80, 80, 0.0, 0.99, 0.0}}
	data, shape := buildOutput(3, anchors)

	dets, err := Decode(data, shape, pp, 320, 160, []string{"a", "b", "c"}, 0.25, 0.45)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(dets) != 1 {
		t.Fatalf("want 1 detection, got %d", len(dets))
	}
	// center x 100 -> x1 = 100-40/... : w=80 in input -> 40 in orig; x1=100-20=80? recompute:
	// x1_input = 200-40=160 ; (160-0)/2 = 80 ; x2_input=240 -> 120
	b := dets[0].Box
	if b.X1 != 80 || b.X2 != 120 {
		t.Errorf("x mapping wrong: %+v", b)
	}
	// y1_input=320-40=280 -> (280-160)/2 = 60 ; y2_input=360 -> (360-160)/2=100
	if b.Y1 != 60 || b.Y2 != 100 {
		t.Errorf("y mapping wrong: %+v", b)
	}
}

func TestDecodeBadShape(t *testing.T) {
	if _, err := Decode([]float32{1, 2, 3}, []int64{3}, identityPP(640), 640, 640, nil, 0.25, 0.45); err == nil {
		t.Fatal("expected error for bad shape")
	}
}
