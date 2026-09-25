package controller

import (
	"errors"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"net/http"

	"github.com/Mr-coo/VleeFruit/backend/internal/inference"
	"github.com/gin-gonic/gin"
)

// InferenceController runs the AI model on an uploaded image.
type InferenceController struct {
	detector *inference.Detector
}

func NewInferenceController(detector *inference.Detector) *InferenceController {
	return &InferenceController{detector: detector}
}

type boxDTO struct {
	X1     float32 `json:"x1"`
	Y1     float32 `json:"y1"`
	X2     float32 `json:"x2"`
	Y2     float32 `json:"y2"`
	Width  float32 `json:"width_px"`
	Height float32 `json:"height_px"`
}

type detectionDTO struct {
	Label      string  `json:"label"`
	ClassID    int     `json:"class_id"`
	Confidence float32 `json:"confidence"`
	Box        boxDTO  `json:"box"`
}

// Analyze runs the model on a single uploaded image and returns detections.
// @Summary      Detect fruit ripeness in an image
// @Description  Accepts a single image (jpeg/png), runs the YOLO model, and returns decoded detections (ripeness label, confidence, and bounding box in original-image pixels).
// @Tags         inference
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image file to analyze"
// @Success      200    {object}  map[string]interface{}
// @Failure      400    {object}  map[string]string
// @Failure      503    {object}  map[string]string
// @Router       /api/v1/analyze [post]
func (ic *InferenceController) Analyze(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'image' file"})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image (expected jpeg or png)"})
		return
	}

	dets, err := ic.detector.Detect(c.Request.Context(), img)
	if err != nil {
		if errors.Is(err, inference.ErrModelNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]detectionDTO, 0, len(dets))
	for _, d := range dets {
		out = append(out, detectionDTO{
			Label:      d.Label,
			ClassID:    d.ClassID,
			Confidence: d.Confidence,
			Box: boxDTO{
				X1: d.Box.X1, Y1: d.Box.Y1, X2: d.Box.X2, Y2: d.Box.Y2,
				Width: d.Box.Width(), Height: d.Box.Height(),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":      len(out),
		"detections": out,
	})
}
