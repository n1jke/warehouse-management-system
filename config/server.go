package config

import (
	"fmt"
	"time"
)

type GRPCConfig struct {
	Host            string        `env:"GRPC_HOST" envDefault:"0.0.0.0"`
	Port            int           `env:"GRPC_PORT" envDefault:"50051"`
	ShutdownTimeout time.Duration `env:"GRPC_SHUTDOWN_TIMEOUT_SEC" envDefault:"10"`
	MaxConnIdle     time.Duration `env:"GRPC_KEEPALIVE_MAX_CONNECTION_IDLE" envDefault:"5m"`
	MaxConnAge      time.Duration `env:"GRPC_KEEPALIVE_MAX_CONNECTION_AGE" envDefault:"30m"`
	KeepAlive       time.Duration `env:"GRPC_KEEPALIVE" envDefault:"10s"`
	RPS             int           `env:"GRPC_LIMIT_RPS" envDefault:"1000"`
	Burst           int           `env:"GRPC_LIMIT_BURST" envDefault:"1500"`
}

func (c *GRPCConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *GRPCConfig) Validate() error {
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("grpc shutdown timeout must be greater than 0")
	}

	if c.KeepAlive <= 0 {
		return fmt.Errorf("grpc keepalive must be greater than 0")
	}

	if c.MaxConnIdle <= 0 {
		return fmt.Errorf("grpc max connection idle must be greater than 0")
	}

	if c.MaxConnAge <= 0 {
		return fmt.Errorf("grpc max connection age must be greater than 0")
	}

	return nil
}
