package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

const (
	addWave = `
		INSERT INTO waves (id, status, max_orders, created_at, closed_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	addWaveOrder = `
		INSERT INTO wave_orders (wave_id, order_id)
		VALUES ($1, $2)
	`

	getWaveByID = `
		SELECT status, max_orders, created_at, closed_at
		FROM waves
		WHERE id = $1
	`

	getWaveOrdersByWaveID = `
		SELECT order_id
		FROM wave_orders
		WHERE wave_id = $1
		ORDER BY order_id ASC
	`

	getWaveOrdersByWaveIDs = `
		SELECT wave_id, order_id
		FROM wave_orders
		WHERE wave_id = ANY($1)
		ORDER BY wave_id, order_id ASC
	`

	updateWave = `
		UPDATE waves
		SET status = $2, closed_at = $3
		WHERE id = $1
	`

	listByStatusBase = "SELECT id, status, max_orders, created_at, closed_at FROM waves WHERE status = $1"

	defaultWaveListLimit = 100
)

type WaveRepo struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewWaveRepo(logger *slog.Logger, db *pgxpool.Pool) *WaveRepo {
	return &WaveRepo{
		logger: logger.With("module", "wave-repo"),
		db:     db,
	}
}

func (r *WaveRepo) Add(ctx context.Context, wave *domain.Wave) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, addWave, wave.ID(), wave.Status(), wave.OrdersCount(), wave.CreatedAt(), wave.ClosedAt())
	if err != nil {
		r.logger.Error("add wave", slog.Any("err", err), slog.Any("wave_id", wave.ID()))
		return err
	}

	if err := r.saveWaveOrders(ctx, wave); err != nil {
		return err
	}

	return nil
}

func (r *WaveRepo) saveWaveOrders(ctx context.Context, wave *domain.Wave) error {
	querier := GetQuerier(ctx, r.db)

	for _, orderID := range wave.Orders() {
		_, err := querier.Exec(ctx, addWaveOrder, wave.ID(), orderID)
		if err != nil {
			r.logger.Error("add wave order", slog.Any("err", err), slog.Any("wave_id", wave.ID()), slog.Any("order_id", orderID))
			return err
		}
	}

	return nil
}

func (r *WaveRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Wave, error) {
	querier := GetQuerier(ctx, r.db)

	row := querier.QueryRow(ctx, getWaveByID, id)

	var (
		status    string
		maxOrders int
		createdAt time.Time
		closedAt  *time.Time
	)

	err := row.Scan(&status, &maxOrders, &createdAt, &closedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}

		r.logger.Error("get wave by id", slog.Any("err", err), slog.Any("wave_id", id))

		return nil, err
	}

	orderIDs, err := r.getWaveOrdersByWaveID(ctx, id)
	if err != nil {
		return nil, err
	}

	return domain.WaveFromExist(id, domain.WaveStatus(status), orderIDs, maxOrders, createdAt, closedAt), nil
}

func (r *WaveRepo) getWaveOrdersByWaveID(ctx context.Context, waveID uuid.UUID) ([]uuid.UUID, error) {
	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getWaveOrdersByWaveID, waveID)
	if err != nil {
		r.logger.Error("get wave orders by wave id", slog.Any("err", err), slog.Any("wave_id", waveID))
		return nil, err
	}
	defer rows.Close()

	var orderIDs []uuid.UUID

	for rows.Next() {
		var orderID uuid.UUID

		if err := rows.Scan(&orderID); err != nil {
			r.logger.Error("scan wave order row", slog.Any("err", err))
			return nil, err
		}

		orderIDs = append(orderIDs, orderID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orderIDs, nil
}

//nolint:funlen // todo: refactor this method to reduce complexity
func (r *WaveRepo) GetByStatus(ctx context.Context, status domain.WaveStatus, limit int, cursor application.Cursor,
) ([]*domain.Wave, error) {
	querier := GetQuerier(ctx, r.db)

	if limit <= 0 {
		limit = defaultWaveListLimit
	}

	query := listByStatusBase
	args := []any{string(status)}

	if !cursor.IsEmpty() {
		query += " AND (created_at, id) > ($" + fmt.Sprintf("%d", len(args)+1) + ", $" + fmt.Sprintf("%d", len(args)+2) + ")"
		args = append(args, cursor.LastCreatedAt, cursor.LastID)
	}

	query += " ORDER BY created_at ASC, id ASC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := querier.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("list waves query", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	type waveRow struct {
		id        uuid.UUID
		status    string
		maxOrders int
		createdAt time.Time
		closedAt  *time.Time
	}

	var rowsData []waveRow

	for rows.Next() {
		var row waveRow

		if err := rows.Scan(&row.id, &row.status, &row.maxOrders, &row.createdAt, &row.closedAt); err != nil {
			r.logger.Error("scan wave row", slog.Any("err", err))
			return nil, err
		}

		rowsData = append(rowsData, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(rowsData) == 0 {
		return nil, nil
	}

	waveIDs := make([]uuid.UUID, len(rowsData))
	for i, r := range rowsData {
		waveIDs[i] = r.id
	}

	ordersMap, err := r.getWaveOrdersMapByWaveIDs(ctx, waveIDs)
	if err != nil {
		return nil, err
	}

	waves := make([]*domain.Wave, 0, len(rowsData))

	for _, row := range rowsData {
		orderIDs := ordersMap[row.id]
		if orderIDs == nil {
			orderIDs = []uuid.UUID{}
		}

		waves = append(waves, domain.WaveFromExist(row.id, domain.WaveStatus(row.status), orderIDs, row.maxOrders, row.createdAt, row.closedAt))
	}

	return waves, nil
}

func (r *WaveRepo) getWaveOrdersMapByWaveIDs(ctx context.Context, waveIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	if len(waveIDs) == 0 {
		return nil, nil
	}

	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getWaveOrdersByWaveIDs, waveIDs)
	if err != nil {
		r.logger.Error("get wave orders by wave ids", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	ordersMap := make(map[uuid.UUID][]uuid.UUID)

	for rows.Next() {
		var waveID, orderID uuid.UUID

		if err := rows.Scan(&waveID, &orderID); err != nil {
			r.logger.Error("scan wave order batch", slog.Any("err", err))
			return nil, err
		}

		ordersMap[waveID] = append(ordersMap[waveID], orderID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ordersMap, nil
}

func (r *WaveRepo) Update(ctx context.Context, wave *domain.Wave) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, updateWave, wave.ID(), wave.Status(), wave.ClosedAt())
	if err != nil {
		r.logger.Error("update wave", slog.Any("err", err), slog.Any("wave_id", wave.ID()))
		return err
	}

	return nil
}
