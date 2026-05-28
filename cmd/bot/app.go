package main

import (
	"log/slog"
	"os"

	"github.com/n1jke/warehouse-management-system/config"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func NewApp() fx.Option {
	return fx.Options(
		fx.Provide(NewLogger),
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		config.Module,
	)
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
