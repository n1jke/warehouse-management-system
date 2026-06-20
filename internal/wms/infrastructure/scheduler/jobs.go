package scheduler

import (
	"context"
	"fmt"
	"log/slog"
)

func (s *Sentinel) relayOutbox(ctx context.Context) {
	if err := s.tx.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.relayOutboxTx(txCtx)
	}); err != nil {
		s.logger.Error("outbox relay failed", slog.Any("err", err))
	}
}

func (s *Sentinel) relayOutboxTx(ctx context.Context) error {
	records, err := s.outboxRepo.FetchPending(ctx, s.cfg.BatchSize)
	if err != nil {
		return fmt.Errorf("fetch pending outbox: %w", err)
	}

	for _, rec := range records {
		if err := s.producer.Publish(ctx, rec); err != nil {
			s.logger.Error("publish outbox record", slog.Any("record_id", rec.EventID), slog.Any("err", err))
			continue
		}

		if err := s.outboxRepo.UpdateStatus(ctx, rec.EventID, nil); err != nil {
			return fmt.Errorf("update outbox status: %w", err)
		}
	}

	return nil
}

func (s *Sentinel) planWaves(ctx context.Context) {
	wave, err := s.wave.CreateWave(ctx, s.cfg.WaveMaxOrders)
	if err != nil {
		s.logger.Error("wave planning failed", slog.Any("err", err))
		return
	}

	if wave != nil {
		s.logger.Info("wave created", slog.Any("wave_id", wave.ID()), slog.Int("orders", len(wave.Orders())))
	}
}

func (s *Sentinel) cleanup(ctx context.Context) {
	deleted, err := s.outboxRepo.Cleanup(ctx, s.cfg.CleanupGap)
	if err != nil {
		s.logger.Error("outbox cleanup failed", slog.Any("err", err))
		return
	}

	if deleted > 0 {
		s.logger.Info("outbox cleanup completed", slog.Int64("deleted", deleted))
	}
}
