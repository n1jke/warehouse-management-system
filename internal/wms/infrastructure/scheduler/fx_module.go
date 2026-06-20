package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"scheduler",
	fx.Provide(
		NewSentinel,
	),
	fx.Invoke(RegisterHooks),
)

func RegisterHooks(lc fx.Lifecycle, s *Sentinel, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			ctx := context.Background()

			err := s.Start(ctx)
			if err != nil {
				logger.Error("start scheduler", slog.Any("err", err))
				return fmt.Errorf("start scheduler: %w", err)
			}

			logger.Info("scheduler started")

			return nil
		},
		OnStop: func(_ context.Context) error {
			logger.Info("scheduler stopping")

			if err := s.Stop(); err != nil {
				logger.Error("stop scheduler", slog.Any("err", err))
				return fmt.Errorf("stop scheduler: %w", err)
			}

			return nil
		},
	})
}
