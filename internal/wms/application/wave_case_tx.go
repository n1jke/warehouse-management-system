package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func (s *WaveService) createWaveTx(ctx context.Context, wave *domain.Wave, maxOrders int) error {
	orders, err := s.orderRepo.GetByStatus(ctx, domain.StatusReserved, maxOrders, Cursor{})
	if err != nil {
		return fmt.Errorf("fetch reserved orders: %w", err)
	}

	for _, order := range orders {
		if wave.IsFull() {
			s.logger.Info("wave is full", slog.Int("orders", len(wave.Orders())), slog.Any("waveID", wave.ID()))
			break
		}

		if err := wave.AddOrder(order.ID()); err != nil {
			return fmt.Errorf("add order to wave: %w", err)
		}

		if err := order.TransitionTo(domain.StatusInWave); err != nil {
			s.logger.Error("transition order to wave", slog.Any("err", err))
			return fmt.Errorf("transition order to in_wave: %w", err)
		}
	}

	if err := s.orderRepo.UpdateStatusBatch(ctx, wave.Orders(), domain.StatusInWave); err != nil {
		s.logger.Error("update orders to in_wave", slog.Any("err", err))
		return fmt.Errorf("update order to in_wave: %w", err)
	}

	if err := s.waveRepo.Add(ctx, wave); err != nil {
		s.logger.Error("save wave", slog.Any("err", err))
		return fmt.Errorf("save wave: %w", err)
	}

	return nil
}

func (s *WaveService) closeWaveTx(ctx context.Context, waveID uuid.UUID) error {
	wave, err := s.waveRepo.GetByID(ctx, waveID)
	if err != nil {
		return fmt.Errorf("get wave by id: %w", err)
	}

	if err := wave.Close(); err != nil {
		return err
	}

	if err := s.waveRepo.Update(ctx, wave); err != nil {
		s.logger.Error("update wave status", slog.Any("err", err))
		return fmt.Errorf("update wave status: %w", err)
	}

	return nil
}

func (s *WaveService) completeWaveTx(ctx context.Context, waveID uuid.UUID) error {
	wave, err := s.waveRepo.GetByID(ctx, waveID)
	if err != nil {
		return fmt.Errorf("get wave by id: %w", err)
	}

	if err := wave.Complete(); err != nil {
		return err
	}

	if err := s.waveRepo.Update(ctx, wave); err != nil {
		s.logger.Error("update wave status", slog.Any("err", err))
		return fmt.Errorf("update wave status: %w", err)
	}

	return nil
}

func (s *WaveService) fetchWaves(ctx context.Context, input ListWavesInput, limit int, cursor Cursor) ([]*domain.Wave, error) {
	if input.Status != nil {
		return s.waveRepo.GetByStatus(ctx, *input.Status, limit, cursor)
	}

	return s.waveRepo.GetByStatus(ctx, domain.WaveStatusOpen, limit, cursor)
}
