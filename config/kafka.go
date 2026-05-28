package config

import (
	"errors"
	"time"

	"github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/kafka"
)

type KafkaConfig struct {
	Brokers           []string      `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
	Topic             string        `env:"KAFKA_TOPIC" envDefault:"order-events"`
	Attempts          int           `env:"KAFKA_ATTEMPTS" envDefault:"3"`
	BatchSize         int           `env:"KAFKA_BATCH_SIZE" envDefault:"100"`
	BatchTimeout      time.Duration `env:"KAFKA_BATCH_TIMEOUT" envDefault:"500ms"`
	SchemaRegistryURL string        `env:"SCHEMA_REGISTRY_URL" envDefault:"http://localhost:8081"`
	Username          string        `env:"KAFKA_USERNAME"`
	Password          string        `env:"KAFKA_PASSWORD"`
}

func ProvideKafkaConfig(cfg *AppConfig) *KafkaConfig {
	return &cfg.Kafka
}

func ToKafkaConfig(cfg *KafkaConfig) *kafka.Config {
	return &kafka.Config{
		Brokers:           cfg.Brokers,
		Topic:             cfg.Topic,
		Attempts:          cfg.Attempts,
		BatchSize:         cfg.BatchSize,
		BatchTimeout:      cfg.BatchTimeout,
		SchemaRegistryURL: cfg.SchemaRegistryURL,
		Username:          cfg.Username,
		Password:          cfg.Password,
	}
}

func (c *KafkaConfig) Validate() error {
	if len(c.Brokers) == 0 {
		return errors.New("kafka brokers are required")
	}

	if c.Topic == "" {
		return errors.New("kafka topic is required")
	}

	if c.Attempts <= 0 {
		return errors.New("kafka attempts must be greater than zero")
	}

	if c.BatchSize <= 0 {
		return errors.New("kafka batch size must be greater than zero")
	}

	if c.SchemaRegistryURL == "" {
		return errors.New("schema registry url is required")
	}

	return nil
}
