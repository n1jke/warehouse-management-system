package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
)

const (
	publishOutbox = `
		INSERT INTO outbox (event_id, event_type, order_id, user_id, status, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	fetchPendingOutbox = `
		SELECT event_id, event_type, order_id, user_id, status, occurred_at
		FROM outbox
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	updateErrorOutbox = `
		UPDATE outbox
		SET retry_count = retry_count + 1, error = $2
		WHERE event_id = $1
	`

	updateSuccessOutbox = `
		UPDATE outbox
		SET processed_at = NOW(), error = NULL
		WHERE event_id = $1
	`

	deleteOldOutbox = `
		DELETE FROM outbox
		WHERE processed_at IS NOT NULL AND processed_at < $1
	`
)

type OutboxRepo struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewOutboxRepo(logger *slog.Logger, db *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{
		logger: logger.With(slog.String("module", "outbox-repo")),
		db:     db,
	}
}

func (o *OutboxRepo) Publish(ctx context.Context, event *application.OrderEvent) error {
	querier := GetQuerier(ctx, o.db)

	_, err := querier.Exec(ctx, publishOutbox, event.EventID, event.EventType, event.OrderID, event.UserID, event.Status, event.OccurredAt)
	if err != nil {
		o.logger.Error("publish outbox", slog.Any("err", err), slog.Any("event_id", event.EventID))
		return fmt.Errorf("publish outbox: %w", err)
	}

	return nil
}

func (o *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]*application.OrderEvent, error) {
	querier := GetQuerier(ctx, o.db)

	rows, err := querier.Query(ctx, fetchPendingOutbox, limit)
	if err != nil {
		o.logger.Error("fetch pending outbox", slog.Any("err", err))
		return nil, fmt.Errorf("fetch pending outbox: %w", err)
	}
	defer rows.Close()

	var records []*application.OrderEvent

	for rows.Next() {
		var rec application.OrderEvent

		err := rows.Scan(&rec.EventID, &rec.EventType, &rec.OrderID, &rec.UserID, &rec.Status, &rec.OccurredAt)
		if err != nil {
			o.logger.Warn("scan outbox row", slog.Any("err", err))
			continue
		}

		records = append(records, &rec)
	}

	if err := rows.Err(); err != nil {
		o.logger.Error("fetch pending outbox rows", slog.Any("err", err))
		return nil, fmt.Errorf("fetch pending outbox rows: %w", err)
	}

	return records, nil
}

func (o *OutboxRepo) UpdateStatus(ctx context.Context, id uuid.UUID, errIn error) error {
	querier := GetQuerier(ctx, o.db)

	if errIn != nil {
		_, err := querier.Exec(ctx, updateErrorOutbox, id, errIn.Error())
		if err != nil {
			o.logger.Error("update outbox error", slog.Any("err", err), slog.Any("event_id", id))
			return fmt.Errorf("update outbox error: %w", err)
		}

		return nil
	}

	_, err := querier.Exec(ctx, updateSuccessOutbox, id)
	if err != nil {
		o.logger.Error("update outbox success", slog.Any("err", err), slog.Any("event_id", id))
		return fmt.Errorf("update outbox success: %w", err)
	}

	return nil
}

func (o *OutboxRepo) Cleanup(ctx context.Context, gap time.Duration) (int64, error) {
	querier := GetQuerier(ctx, o.db)
	cut := time.Now().Add(-gap)

	tag, err := querier.Exec(ctx, deleteOldOutbox, cut)
	if err != nil {
		o.logger.Error("cleanup outbox", slog.Any("err", err))
		return 0, fmt.Errorf("cleanup outbox: %w", err)
	}

	return tag.RowsAffected(), nil
}
