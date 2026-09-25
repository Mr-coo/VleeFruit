package controller

import (
	"net/http"
	"strings"

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

// ImageController handles image uploads.
type ImageController struct{}

func NewImageController() *ImageController {
	return &ImageController{}
}

// Upload accepts a single image file via multipart/form-data (field "image").
// @Summary      Upload an image
// @Description  Accepts a single image file (jpeg, png or webp) in the "image" form field.
// @Tags         images
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image file to upload"
// @Success      200    {object}  map[string]interface{}
// @Failure      400    {object}  map[string]string
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

	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedImageTypes[strings.ToLower(contentType)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type, expected jpeg/png/webp"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename":     fileHeader.Filename,
		"size":         fileHeader.Size,
		"content_type": contentType,
	})
}
