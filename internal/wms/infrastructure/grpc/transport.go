package grpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
)

type Config struct {
	Addr            string
	ShutdownTimeout time.Duration
	MaxConnIdle     time.Duration
	MaxConnAge      time.Duration
	KeepAlive       time.Duration
	RPS             int
	Burst           int
}

type RunningServer struct {
	server *grpc.Server
	addr   string
}

func NewRunningServer(addr string, opts []grpc.ServerOption, register func(*grpc.Server)) *RunningServer {
	svr := grpc.NewServer(opts...)
	register(svr)

	return &RunningServer{
		server: svr,
		addr:   addr,
	}
}

func (s *RunningServer) Start(_ context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	return s.server.Serve(lis)
}

func (s *RunningServer) Stop(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	}
}
