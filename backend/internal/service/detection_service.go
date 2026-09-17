package service

import (
	"context"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
)

// DetectionService exposes stored detection results for the query API.
type DetectionService struct {
	detectionRepo *repository.DetectionRepository
	imageRepo     *repository.ImageRepository
}

func NewDetectionService(detectionRepo *repository.DetectionRepository, imageRepo *repository.ImageRepository) *DetectionService {
	return &DetectionService{detectionRepo: detectionRepo, imageRepo: imageRepo}
}

func (s *DetectionService) ListResults(ctx context.Context, limit int) ([]domain.DetectionResult, error) {
	return s.detectionRepo.List(ctx, limit)
}

func (s *DetectionService) ListImages(ctx context.Context, limit int) ([]domain.Image, error) {
	return s.imageRepo.List(ctx, limit)
}
