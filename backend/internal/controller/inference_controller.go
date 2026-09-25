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
	runner inference.Runner
}

func NewInferenceController(runner inference.Runner) *InferenceController {
	return &InferenceController{runner: runner}
}

// Analyze runs the model on a single uploaded image.
// @Summary      Run model inference on an image
// @Description  Accepts a single image (jpeg/png) and runs the ONNX model, returning the raw output tensor shape. Model-specific decoding (boxes, ripeness, defects) is applied by the service layer.
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

	out, err := ic.runner.Infer(c.Request.Context(), img)
	if err != nil {
		if errors.Is(err, inference.ErrModelNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output_shape": out.Shape,
		"output_len":   len(out.Data),
	})
}
