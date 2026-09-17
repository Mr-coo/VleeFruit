// Package imageutil resizes and re-encodes images before they are stored, to
// keep object-storage usage small (important on free tiers).
package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	// Register decoders so image.Decode handles JPEG and PNG input.
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

// Processed is the result of resizing an image.
type Processed struct {
	Data        []byte
	Width       int
	Height      int
	ContentType string
}

// ResizeJPEG decodes src, scales it so its longest side is at most maxDim
// (never upscales), and re-encodes as JPEG at the given quality (1-100).
func ResizeJPEG(src []byte, maxDim, quality int) (Processed, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return Processed{}, fmt.Errorf("decode image: %w", err)
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := scaledDims(w, h, maxDim)

	out := img
	if nw != w || nh != h {
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		out = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: quality}); err != nil {
		return Processed{}, fmt.Errorf("encode jpeg: %w", err)
	}

	return Processed{
		Data:        buf.Bytes(),
		Width:       nw,
		Height:      nh,
		ContentType: "image/jpeg",
	}, nil
}

// scaledDims returns dimensions scaled to fit within maxDim on the longest
// side, preserving aspect ratio and never upscaling.
func scaledDims(w, h, maxDim int) (int, int) {
	if maxDim <= 0 || (w <= maxDim && h <= maxDim) {
		return w, h
	}
	if w >= h {
		return maxDim, int(float64(h) * float64(maxDim) / float64(w))
	}
	return int(float64(w) * float64(maxDim) / float64(h)), maxDim
}
