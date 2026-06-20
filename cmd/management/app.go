package main

import (
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/n1jke/warehouse-management-system/config"
	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/grpc"
	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/kafka"
	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/repository"
	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/scheduler"
)

func NewApp() fx.Option {
	return fx.Options(
		fx.Provide(NewLogger),
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		config.Module,
		application.Module,
		repository.Module,
		kafka.Module,
		grpc.Module,
		scheduler.Module,
	)
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
