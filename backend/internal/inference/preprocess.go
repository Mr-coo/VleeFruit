package inference

import (
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"
)

// PreprocessResult holds the model input tensor plus the letterbox geometry,
// which is needed later to map model output coordinates back to the original
// image (postprocessing is model-specific and left to the caller).
type PreprocessResult struct {
	// Data is the normalized input tensor in NCHW order (1x3xSizexSize), RGB,
	// values scaled to [0,1].
	Data []float32
	// Scale is the resize factor applied to the original image.
	Scale float64
	// PadX/PadY are the letterbox padding offsets in pixels.
	PadX, PadY int
	// Size is the square input dimension used.
	Size int
}

// Preprocess letterboxes img into a SizexSize square (preserving aspect ratio,
// padding with gray 114) and converts it to a normalized NCHW float32 tensor.
// This matches the standard YOLO input convention.
func Preprocess(img image.Image, size int) PreprocessResult {
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()

	scale := math.Min(float64(size)/float64(srcW), float64(size)/float64(srcH))
	newW := int(math.Round(float64(srcW) * scale))
	newH := int(math.Round(float64(srcH) * scale))
	padX := (size - newW) / 2
	padY := (size - newH) / 2

	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{114, 114, 114, 255}}, image.Point{}, xdraw.Src)

	dstRect := image.Rect(padX, padY, padX+newW, padY+newH)
	xdraw.CatmullRom.Scale(canvas, dstRect, img, b, xdraw.Over, nil)

	data := make([]float32, 3*size*size)
	plane := size * size
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := canvas.PixOffset(x, y)
			idx := y*size + x
			data[idx] = float32(canvas.Pix[i]) / 255.0           // R plane
			data[plane+idx] = float32(canvas.Pix[i+1]) / 255.0   // G plane
			data[2*plane+idx] = float32(canvas.Pix[i+2]) / 255.0 // B plane
		}
	}

	return PreprocessResult{Data: data, Scale: scale, PadX: padX, PadY: padY, Size: size}
}
