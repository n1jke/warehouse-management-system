package config

import (
	"errors"
	"time"
)

type SchedulerConfig struct {
	BatchSize       int           `env:"SCHEDULER_BATCH_SIZE"       envDefault:"50"`
	OutboxInterval  time.Duration `env:"SCHEDULER_OUTBOX_INTERVAL"  envDefault:"5s"`
	WaveInterval    time.Duration `env:"SCHEDULER_WAVE_INTERVAL"    envDefault:"30s"`
	WaveMaxOrders   int           `env:"SCHEDULER_WAVE_MAX_ORDERS"  envDefault:"50"`
	CleanupInterval time.Duration `env:"SCHEDULER_CLEANUP_INTERVAL" envDefault:"1h"`
	CleanupGap      time.Duration `env:"SCHEDULER_CLEANUP_GAP"     envDefault:"168h"`
}

func (c *SchedulerConfig) Validate() error {
	if c.BatchSize <= 0 {
		return errors.New("SCHEDULER_BATCH_SIZE must be > 0")
	}

	if c.OutboxInterval <= 0 {
		return errors.New("SCHEDULER_OUTBOX_INTERVAL must be > 0")
	}

	if c.WaveInterval <= 0 {
		return errors.New("SCHEDULER_WAVE_INTERVAL must be > 0")
	}

	if c.WaveMaxOrders <= 0 {
		return errors.New("SCHEDULER_WAVE_MAX_ORDERS must be > 0")
	}

	if c.CleanupInterval <= 0 {
		return errors.New("SCHEDULER_CLEANUP_INTERVAL must be > 0")
	}

	if c.CleanupGap <= 0 {
		return errors.New("SCHEDULER_CLEANUP_GAP must be > 0")
	}

	return nil
}
