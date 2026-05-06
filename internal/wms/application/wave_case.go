package application

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type ListWavesInput struct {
	Status    *domain.WaveStatus
	PageToken string
	PageSize  int
}

type ListWavesOutput struct {
	Waves         []domain.Wave
	NextPageToken string
}

type WaveService struct {
	logger    *slog.Logger
	waveRepo  WaveRepository
	orderRepo OrderRepository
}

func NewWaveService(logger *slog.Logger, waveRepo WaveRepository, orderRepo OrderRepository) *WaveService {
	return &WaveService{
		logger:    logger.With(slog.String("module", "wave-service")),
		waveRepo:  waveRepo,
		orderRepo: orderRepo,
	}
}

// ErrNotFound
func (s *WaveService) GetWave(ctx context.Context, waveID uuid.UUID) (*domain.Wave, error) {
	return nil, nil
}

// waves with pagination
func (s *WaveService) ListWaves(ctx context.Context, input ListWavesInput) (*ListWavesOutput, error) {
	return nil, nil
}
