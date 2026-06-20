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
	addOrder = `
		INSERT INTO orders (id, user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	addOrderItem = `
		INSERT INTO order_items (order_id, sku, quantity)
		VALUES ($1, $2, $3)
	`

	getOrderByID = `
		SELECT user_id, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	getItemsByOrderID = `
		SELECT sku, quantity
		FROM order_items
		WHERE order_id = $1
	`

	getItemsByOrderIDs = `
		SELECT order_id, sku, quantity
		FROM order_items
		WHERE order_id = ANY($1)
	`

	updateOrder = `
		UPDATE orders
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	deleteOrderItems = `
		DELETE FROM order_items
		WHERE order_id = $1
	`

	updateOrderStatusBatch = `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = ANY($1)
	`

	deleteOrder = `
		DELETE FROM orders
		WHERE id = $1
	`

	defaultOrderLimit = 100
)

var orderListByUserID = "SELECT id, user_id, status, created_at, updated_at FROM orders WHERE user_id = $1"

var orderListByStatus = "SELECT id, user_id, status, created_at, updated_at FROM orders WHERE status = $1"

var orderListByStatusAndUserID = "SELECT id, user_id, status, created_at, updated_at FROM orders WHERE user_id = $1 AND status = $2"

type OrderRepo struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewOrderRepo(logger *slog.Logger, db *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{
		logger: logger.With("module", "order-repo"),
		db:     db,
	}
}

func (r *OrderRepo) Add(ctx context.Context, order *domain.Order) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, addOrder, order.ID(), order.UserID(), order.Status(), order.CreatedAt(), order.UpdatedAt())
	if err != nil {
		r.logger.Error("add order", slog.Any("err", err), slog.Any("order_id", order.ID()))
		return err
	}

	for _, item := range order.Items() {
		_, err := querier.Exec(ctx, addOrderItem, order.ID(), item.SKU, item.Quantity)
		if err != nil {
			r.logger.Error("add order item", slog.Any("err", err), slog.Any("order_id", order.ID()), slog.String("sku", item.SKU))
			return err
		}
	}

	return nil
}

func (r *OrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	querier := GetQuerier(ctx, r.db)

	row := querier.QueryRow(ctx, getOrderByID, id)

	var (
		userID               int64
		status               string
		createdAt, updatedAt time.Time
	)

	err := row.Scan(&userID, &status, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}

		r.logger.Error("get order by id", slog.Any("err", err), slog.Any("order_id", id))

		return nil, err
	}

	items, err := r.getItemsByOrderID(ctx, id)
	if err != nil {
		return nil, err
	}

	return domain.OrderFromExist(id, userID, domain.OrderStatus(status), items, createdAt, updatedAt), nil
}

func (r *OrderRepo) getItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getItemsByOrderID, orderID)
	if err != nil {
		r.logger.Error("get items by order id", slog.Any("err", err), slog.Any("order_id", orderID))
		return nil, err
	}
	defer rows.Close()

	var items []domain.OrderItem

	for rows.Next() {
		var item domain.OrderItem

		if err := rows.Scan(&item.SKU, &item.Quantity); err != nil {
			r.logger.Error("scan order item", slog.Any("err", err))
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *OrderRepo) GetByUserID(ctx context.Context, userID int64, limit int, cursor application.Cursor) ([]*domain.Order, error) {
	return r.listOrders(ctx, userIDFilter(userID), limit, cursor)
}

func (r *OrderRepo) GetByStatus(ctx context.Context, status domain.OrderStatus, limit int, cursor application.Cursor,
) ([]*domain.Order, error) {
	return r.listOrders(ctx, statusFilter(status), limit, cursor)
}

func (r *OrderRepo) GetByStatusAndUserID(ctx context.Context, userID int64, status domain.OrderStatus, limit int,
	cursor application.Cursor,
) ([]*domain.Order, error) {
	return r.listOrders(ctx, statusAndUserIDFilter(userID, status), limit, cursor)
}

type orderFilter struct {
	query string
	args  []any
}

func userIDFilter(userID int64) orderFilter {
	return orderFilter{
		query: orderListByUserID,
		args:  []any{userID},
	}
}

func statusFilter(status domain.OrderStatus) orderFilter {
	return orderFilter{
		query: orderListByStatus,
		args:  []any{string(status)},
	}
}

func statusAndUserIDFilter(userID int64, status domain.OrderStatus) orderFilter {
	return orderFilter{
		query: orderListByStatusAndUserID,
		args:  []any{userID, string(status)},
	}
}

type orderRow struct {
	id        uuid.UUID
	userID    int64
	status    string
	createdAt time.Time
	updatedAt time.Time
}

func (r *OrderRepo) listOrders(ctx context.Context, f orderFilter, limit int, cursor application.Cursor) ([]*domain.Order, error) {
	querier := GetQuerier(ctx, r.db)

	query := f.query
	args := make([]any, 0, len(f.args)+3)
	args = append(args, f.args...)

	if !cursor.IsEmpty() {
		query += " AND (created_at, id) > ($" + fmt.Sprintf("%d", len(args)+1) + ", $" + fmt.Sprintf("%d", len(args)+2) + ")"
		args = append(args, cursor.LastCreatedAt, cursor.LastID)
	}

	if limit <= 0 {
		limit = defaultOrderLimit
	}

	query += " ORDER BY created_at ASC, id ASC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := querier.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("list orders query", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	var rowsData []orderRow

	for rows.Next() {
		var row orderRow

		if err := rows.Scan(&row.id, &row.userID, &row.status, &row.createdAt, &row.updatedAt); err != nil {
			r.logger.Error("scan order row", slog.Any("err", err))
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

	orderIDs := make([]uuid.UUID, len(rowsData))
	for i, r := range rowsData {
		orderIDs[i] = r.id
	}

	itemsMap, err := r.getItemsMapByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}

	orders := make([]*domain.Order, 0, len(rowsData))

	for _, row := range rowsData {
		items := itemsMap[row.id]
		if items == nil {
			items = []domain.OrderItem{}
		}

		orders = append(orders, domain.OrderFromExist(row.id, row.userID, domain.OrderStatus(row.status), items, row.createdAt, row.updatedAt))
	}

	return orders, nil
}

func (r *OrderRepo) getItemsMapByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]domain.OrderItem, error) {
	if len(orderIDs) == 0 {
		return nil, nil
	}

	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getItemsByOrderIDs, orderIDs)
	if err != nil {
		r.logger.Error("get items by order ids", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	itemsMap := make(map[uuid.UUID][]domain.OrderItem)

	for rows.Next() {
		var (
			orderID uuid.UUID
			item    domain.OrderItem
		)

		if err := rows.Scan(&orderID, &item.SKU, &item.Quantity); err != nil {
			r.logger.Error("scan order item batch", slog.Any("err", err))
			return nil, err
		}

		itemsMap[orderID] = append(itemsMap[orderID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return itemsMap, nil
}

func (r *OrderRepo) Update(ctx context.Context, order *domain.Order) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, updateOrder, order.ID(), order.Status(), order.UpdatedAt())
	if err != nil {
		r.logger.Error("update order", slog.Any("err", err), slog.Any("order_id", order.ID()))
		return err
	}

	_, err = querier.Exec(ctx, deleteOrderItems, order.ID())
	if err != nil {
		r.logger.Error("delete order items", slog.Any("err", err), slog.Any("order_id", order.ID()))
		return err
	}

	for _, item := range order.Items() {
		_, err := querier.Exec(ctx, addOrderItem, order.ID(), item.SKU, item.Quantity)
		if err != nil {
			r.logger.Error("re-insert order item", slog.Any("err", err), slog.Any("order_id", order.ID()), slog.String("sku", item.SKU))
			return err
		}
	}

	return nil
}

func (r *OrderRepo) UpdateStatusBatch(ctx context.Context, orderIDs []uuid.UUID, status domain.OrderStatus) error {
	if len(orderIDs) == 0 {
		return nil
	}

	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, updateOrderStatusBatch, orderIDs, string(status))
	if err != nil {
		r.logger.Error("update order status batch", slog.Any("err", err), slog.Any("status", status))
	}

	return err
}

func (r *OrderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, deleteOrderItems, id)
	if err != nil {
		r.logger.Error("delete order items", slog.Any("err", err), slog.Any("order_id", id))
		return err
	}

	_, err = querier.Exec(ctx, deleteOrder, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}

		r.logger.Error("delete order", slog.Any("err", err), slog.Any("order_id", id))

		return err
	}

	return nil
}
