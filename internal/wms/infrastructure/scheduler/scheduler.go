package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
)

type OutboxRepository interface {
	FetchPending(ctx context.Context, limit int) ([]*application.OrderEvent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, errIn error) error
	Cleanup(ctx context.Context, gap time.Duration) (int64, error)
}

type Producer interface {
	Publish(ctx context.Context, event *application.OrderEvent) error
}

type Transactor interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type Config struct {
	BatchSize       int
	WaveMaxOrders   int
	OutboxInterval  time.Duration
	WaveInterval    time.Duration
	CleanupInterval time.Duration
	CleanupGap      time.Duration
}

type Sentinel struct {
	logger     *slog.Logger
	scheduler  gocron.Scheduler
	tx         Transactor
	outboxRepo OutboxRepository
	wave       *application.WaveService
	producer   Producer
	cfg        Config
}

func NewSentinel(logger *slog.Logger, cfg Config, tx Transactor, outboxRepo OutboxRepository,
	waveService *application.WaveService, messageProducer Producer,
) (*Sentinel, error) {
	s, err := gocron.NewScheduler(
		gocron.WithGlobalJobOptions(
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(func(_ uuid.UUID, jobName string) {
					logger.Info("job started", slog.String("job", jobName))
				}),
				gocron.AfterJobRunsWithError(func(_ uuid.UUID, jobName string, err error) {
					logger.Error("job failed", slog.String("job", jobName), slog.Any("err", err))
				}),
				gocron.AfterJobRuns(func(_ uuid.UUID, jobName string) {
					logger.Info("job completed", slog.String("job", jobName))
				}),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	return &Sentinel{
		logger:     logger.With(slog.String("module", "scheduler")),
		scheduler:  s,
		cfg:        cfg,
		tx:         tx,
		outboxRepo: outboxRepo,
		wave:       waveService,
		producer:   messageProducer,
	}, nil
}

func (s *Sentinel) Start(ctx context.Context) error {
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(s.cfg.OutboxInterval),
		gocron.NewTask(s.relayOutbox, ctx),
		gocron.WithName("outbox-relay"),
		gocron.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("register outbox relay job: %w", err)
	}

	_, err = s.scheduler.NewJob(
		gocron.DurationJob(s.cfg.WaveInterval),
		gocron.NewTask(s.planWaves, ctx),
		gocron.WithName("wave-planner"),
		gocron.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("register wave planner job: %w", err)
	}

	_, err = s.scheduler.NewJob(
		gocron.DurationJob(s.cfg.CleanupInterval),
		gocron.NewTask(s.cleanup, ctx),
		gocron.WithName("cleanup-outbox"),
		gocron.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("register cleanup job: %w", err)
	}

	s.scheduler.Start()

	return nil
}

func (s *Sentinel) Stop() error {
	return s.scheduler.Shutdown()
}
