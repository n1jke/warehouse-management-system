package grpc

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/n1jke/warehouse-management-system/internal/api/proto/wms"
)

var Module = fx.Module(
	"grpc-server",
	fx.Provide(
		NewServer,
		NewTransport,
	),
	fx.Invoke(RegisterLifecycle),
)

func NewTransport(cfg Config, impl *Server) *RunningServer {
	opt := make([]grpc.ServerOption, 0, 2)
	opt = append(opt, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: cfg.MaxConnIdle,
		MaxConnectionAge:  cfg.MaxConnAge,
		Timeout:           cfg.KeepAlive,
	}), grpc.ChainUnaryInterceptor(
		UnaryLimitInterceptor(cfg.RPS, cfg.Burst),
	))

	return NewRunningServer(cfg.Addr, opt, func(s *grpc.Server) {
		wms.RegisterUserServiceServer(s, impl)
		wms.RegisterOrderServiceServer(s, impl)
		wms.RegisterWaveServiceServer(s, impl)
		reflection.Register(s) // grpcurl testing
	})
}

func RegisterLifecycle(lc fx.Lifecycle, logger *slog.Logger, runner *RunningServer, cfg Config) {
	var cancel context.CancelFunc

	done := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			var ctx context.Context

			ctx, cancel = context.WithCancel(context.Background())

			go func() {
				defer close(done)

				if err := runner.Start(ctx); err != nil {
					logger.Error("grpc server crashed", slog.Any("err", err))
					cancel()
				}
			}()

			logger.Info("grpc server started")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("grpc server stopping")

			ctxShutdown, cancelShutdown := context.WithTimeout(ctx, cfg.ShutdownTimeout)
			defer cancelShutdown()

			if err := runner.Stop(ctxShutdown); err != nil {
				logger.Error("grpc server graceful stop failed", slog.Any("err", err))
			}

			cancel()

			select {
			case <-done:
				logger.Info("grpc server stopped")
			case <-ctxShutdown.Done():
				logger.Warn("grpc server stop deadline exceeded", slog.Any("err", ctxShutdown.Err()))
			}

			return nil
		},
	})
}
