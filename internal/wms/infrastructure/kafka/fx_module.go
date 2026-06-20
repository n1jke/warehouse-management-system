package kafka

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/scheduler"
)

var Module = fx.Module(
	"kafka",
	fx.Provide(
		NewProducer,
		func(p *Producer) scheduler.Producer { return p },
	),
	fx.Invoke(RegisterLifecycle),
)

func RegisterLifecycle(lc fx.Lifecycle, p *Producer, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(_ context.Context) error {
			logger.Info("stopping kafka producer")
			return p.Close()
		},
	})
}
