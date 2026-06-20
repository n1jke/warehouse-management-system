package config

import (
	"github.com/caarlos0/env/v11"
)

type AppConfig struct {
	DB        DatabaseConfig
	Scheduler SchedulerConfig
	Kafka     KafkaConfig
	GRPC      GRPCConfig
	Bot       BotConfig
}

func (c *AppConfig) Validate() error {
	if err := c.Scheduler.Validate(); err != nil {
		return err
	}

	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	return nil
}

func LoadConfig() (*AppConfig, error) {
	cfg := &AppConfig{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
