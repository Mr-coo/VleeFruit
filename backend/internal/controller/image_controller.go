package controller

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Mr-coo/VleeFruit/backend/internal/llm"
	"github.com/gin-gonic/gin"
)

// maxUploadBytes caps the accepted image size (10 MB).
const maxUploadBytes = 10 << 20

// allowedImageTypes is the set of accepted image content types.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// ImageController handles image uploads and forwards them to the vision LLM.
type ImageController struct {
	llm *llm.Client
}

func NewImageController(client *llm.Client) *ImageController {
	return &ImageController{llm: client}
}

// Upload accepts a single image file via multipart/form-data (field "image")
// and returns the vision LLM's analysis of it.
// @Summary      Upload and analyze an image with the vision LLM
// @Description  Accepts a single image file (jpeg, png or webp) in the "image" form field and returns a natural-language analysis from the Gemini vision model.
// @Tags         images
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image file to upload"
// @Success      200    {object}  map[string]interface{}
// @Failure      400    {object}  map[string]string
// @Failure      503    {object}  map[string]string
// @Router       /api/v1/images [post]
func (ic *ImageController) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'image' file"})
		return
	}

	if fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 10MB)"})
		return
	}

	contentType := strings.ToLower(fileHeader.Header.Get("Content-Type"))
	if !allowedImageTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type, expected jpeg/png/webp"})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read uploaded file"})
		return
	}

	analysis, err := ic.llm.Analyze(c.Request.Context(), data, contentType)
	if err != nil {
		if errors.Is(err, llm.ErrNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename":     fileHeader.Filename,
		"size":         fileHeader.Size,
		"content_type": contentType,
		"analysis":     analysis,
	})
}
