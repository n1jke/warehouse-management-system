package application

import (
	"context"
	"errors"
	"fmt"
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
	tx        Transactor
	waveRepo  WaveRepository
	orderRepo OrderRepository
}

func NewWaveService(logger *slog.Logger, tx Transactor, waveRepo WaveRepository, orderRepo OrderRepository) *WaveService {
	return &WaveService{
		logger:    logger.With(slog.String("module", "wave-service")),
		tx:        tx,
		waveRepo:  waveRepo,
		orderRepo: orderRepo,
	}
}

func (s *WaveService) GetWave(ctx context.Context, waveID uuid.UUID) (*domain.Wave, error) {
	wave, err := s.waveRepo.GetByID(ctx, waveID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrWaveNotFound
		}

		s.logger.Error("get wave by id", slog.Any("waveID", waveID), slog.Any("err", err))

		return nil, fmt.Errorf("get wave by id: %w", err)
	}

	return wave, nil
}

func (s *WaveService) ListWaves(ctx context.Context, input ListWavesInput) (*ListWavesOutput, error) {
	s.logger.Info("list waves start", slog.Any("status", input.Status), slog.Int("page_size", input.PageSize))

	cursor, err := DecodeCursor(input.PageToken)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	limit := input.PageSize

	waves, err := s.fetchWaves(ctx, input, limit+1, cursor)
	if err != nil {
		s.logger.Error("list waves fetch", slog.Any("err", err))
		return nil, fmt.Errorf("list waves: %w", err)
	}

	nextPageToken := ""

	if len(waves) == limit+1 {
		waves = waves[:limit]

		last := waves[len(waves)-1]

		token, err := EncodeCursor(Cursor{
			LastCreatedAt: last.CreatedAt(),
			LastID:        last.ID(),
		})
		if err != nil {
			return nil, fmt.Errorf("encode cursor: %w", err)
		}

		nextPageToken = token
	}

	result := make([]domain.Wave, 0, len(waves))
	for _, wave := range waves {
		result = append(result, *wave)
	}

	s.logger.Info("list waves end", slog.Int("count", len(waves)))

	return &ListWavesOutput{
		Waves:         result,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *WaveService) CreateWave(ctx context.Context, maxOrders int) (*domain.Wave, error) {
	wave, err := domain.NewWave(maxOrders)
	if err != nil {
		return nil, fmt.Errorf("create wave: %w", err)
	}

	err = s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.createWaveTx(ctx, wave, maxOrders)
	})
	if err != nil {
		s.logger.Error("create wave tx", slog.Any("err", err))
		return nil, fmt.Errorf("create wave tx: %w", err)
	}

	return wave, nil
}

func (s *WaveService) CloseWave(ctx context.Context, waveID uuid.UUID) error {
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.closeWaveTx(ctx, waveID)
	})
	if err != nil {
		s.logger.Error("close wave tx", slog.Any("waveID", waveID), slog.Any("err", err))
		return fmt.Errorf("close wave tx: %w", err)
	}

	return nil
}

func (s *WaveService) CompleteWave(ctx context.Context, waveID uuid.UUID) error {
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.completeWaveTx(ctx, waveID)
	})
	if err != nil {
		s.logger.Error("complete wave tx", slog.Any("waveID", waveID), slog.Any("err", err))
		return fmt.Errorf("complete wave tx: %w", err)
	}

	return nil
}
