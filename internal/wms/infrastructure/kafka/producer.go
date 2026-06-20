package kafka

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
)

type Config struct {
	Brokers           []string
	Topic             string
	Attempts          int
	BatchSize         int
	BatchTimeout      time.Duration
	SchemaRegistryURL string
	Username          string
	Password          string
}

type Producer struct {
	logger *slog.Logger
	writer *kafka.Writer
	schema *srclient.Schema
}

func NewProducer(logger *slog.Logger, cfg *Config) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka: no brokers configured")
	}

	transport := &kafka.Transport{}
	if cfg.Username != "" && cfg.Password != "" {
		transport.SASL = plain.Mechanism{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.Topic,
		MaxAttempts:            cfg.Attempts,
		BatchSize:              cfg.BatchSize,
		BatchTimeout:           cfg.BatchTimeout,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: false,
		Transport:              transport,
	}

	schemaClient := srclient.CreateSchemaRegistryClient(cfg.SchemaRegistryURL)

	schema, err := schemaClient.GetLatestSchema(cfg.Topic + "-value")
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to fetch schema for subject %s: %w", cfg.Topic+"-value", err)
	}

	return &Producer{
		logger: logger.With(slog.String("module", "kafka-producer")),
		writer: writer,
		schema: schema,
	}, nil
}

func (p *Producer) Publish(ctx context.Context, event *application.OrderEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	native := mapOrderEventToAvro(event)

	data, err := p.schema.Codec().BinaryFromNative(nil, native)
	if err != nil {
		return fmt.Errorf("kafka: failed to encode avro payload: %w", err)
	}

	payload := make([]byte, 5+len(data))
	payload[0] = 0
	binary.BigEndian.PutUint32(payload[1:5], uint32(p.schema.ID())) //nolint:gosec // schema ID from SR is always uint32
	copy(payload[5:], data)

	msg := kafka.Message{
		Key:   []byte(event.OrderID.String()),
		Value: payload,
	}

	if err = p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka: write message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func mapOrderEventToAvro(event *application.OrderEvent) map[string]any {
	return map[string]any{
		"event_id":    event.EventID.String(),
		"event_type":  string(event.EventType),
		"order_id":    event.OrderID.String(),
		"user_id":     event.UserID,
		"status":      string(event.Status),
		"occurred_at": event.OccurredAt.UnixMilli(),
	}
}
