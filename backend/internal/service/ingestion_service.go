package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/detection"
	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
	"github.com/Mr-coo/VleeFruit/backend/internal/storage"
	"github.com/Mr-coo/VleeFruit/backend/pkg/imageutil"
)

// Tuning for the free-tier image budget.
const (
	maxImageDim  = 1024
	jpegQuality  = 80
)

// IngestionService runs the image pipeline: resize -> store -> detect ->
// persist. It returns the detection result so callers can reply to the device.
type IngestionService struct {
	store     storage.Storage
	detector  detection.Detector
	imageRepo *repository.ImageRepository
}

func NewIngestionService(store storage.Storage, detector detection.Detector, imageRepo *repository.ImageRepository) *IngestionService {
	return &IngestionService{store: store, detector: detector, imageRepo: imageRepo}
}

// IngestImage processes one raw image from a device and returns its result.
func (s *IngestionService) IngestImage(ctx context.Context, deviceID string, raw []byte) (*domain.DetectionResult, error) {
	processed, err := imageutil.ResizeJPEG(raw, maxImageDim, jpegQuality)
	if err != nil {
		return nil, fmt.Errorf("process image: %w", err)
	}

	key, err := objectKey(deviceID)
	if err != nil {
		return nil, err
	}

	if err := s.store.Put(ctx, key, bytes.NewReader(processed.Data), int64(len(processed.Data)), processed.ContentType); err != nil {
		return nil, fmt.Errorf("store image: %w", err)
	}
	slog.DebugContext(ctx, "image stored",
		"device_id", deviceID, "object_key", key, "size_bytes", len(processed.Data),
		"width", processed.Width, "height", processed.Height)

	result, err := s.detector.Detect(ctx, processed.Data, processed.ContentType)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}

	img := &domain.Image{
		DeviceID:    deviceID,
		ObjectKey:   key,
		Bucket:      s.store.Bucket(),
		ContentType: processed.ContentType,
		SizeBytes:   int64(len(processed.Data)),
		Width:       processed.Width,
		Height:      processed.Height,
		CapturedAt:  time.Now(),
		Detection: &domain.DetectionResult{
			Ripeness:     result.Ripeness,
			Confidence:   result.Confidence,
			ModelVersion: result.ModelVersion,
		},
	}
	if err := s.imageRepo.Create(ctx, img); err != nil {
		return nil, fmt.Errorf("persist image: %w", err)
	}

	return img.Detection, nil
}

// objectKey builds a per-device, collision-resistant storage key.
func objectKey(deviceID string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate object key: %w", err)
	}
	return fmt.Sprintf("%s/%s.jpg", deviceID, hex.EncodeToString(buf)), nil
}
