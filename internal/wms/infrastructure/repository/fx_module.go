package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/n1jke/warehouse-management-system/config"
	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/scheduler"
)

var Module = fx.Module(
	"mws-repositories",
	fx.Provide(
		ProvidePool,
		fx.Annotate(
			NewTxChain,
			fx.As(new(application.Transactor)),
		),
		fx.Annotate(
			NewUserRepo,
			fx.As(new(application.UserRepository)),
		),
		fx.Annotate(
			NewOutboxRepo,
			fx.As(new(application.EventPublisher)),
			fx.As(new(scheduler.OutboxRepository)),
		),
		fx.Annotate(
			NewOrderRepo,
			fx.As(new(application.OrderRepository)),
		),
		fx.Annotate(
			NewStockRepo,
			fx.As(new(application.StockRepository)),
		),
		fx.Annotate(
			NewWaveRepo,
			fx.As(new(application.WaveRepository)),
		),
	),
	fx.Invoke(RegisterLifecycle),
)

// todo: work with config & separate from config/
func ProvidePool(cfg *config.AppConfig) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.DB.ConnectionString())
	if err != nil {
		return nil, err
	}

	connConfig.MaxConns = 5
	connConfig.MinConns = 1
	connConfig.MaxConnIdleTime = 500 * time.Millisecond
	connConfig.MaxConnLifetime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), connConfig)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func RegisterLifecycle(lc fx.Lifecycle, pool *pgxpool.Pool) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})
}
