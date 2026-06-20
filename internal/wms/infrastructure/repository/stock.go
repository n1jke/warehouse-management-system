package repository

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

const (
	getStocksBySKUs = `
		SELECT sku, total_quantity
		FROM stocks
		WHERE sku = ANY($1::text[])
	`

	getReservationsBySKUs = `
		SELECT sku, order_id, reserved_qty, backorder_qty
		FROM reservations
		WHERE sku = ANY($1::text[])
	`

	upsertStock = `
		INSERT INTO stocks (sku, total_quantity)
		VALUES ($1, $2)
		ON CONFLICT (sku) DO UPDATE SET total_quantity = EXCLUDED.total_quantity
	`

	deleteReservationsBySKU = `
		DELETE FROM reservations
		WHERE sku = $1
	`

	addReservation = `
		INSERT INTO reservations (order_id, sku, reserved_qty, backorder_qty)
		VALUES ($1, $2, $3, $4)
	`
)

type StockRepo struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewStockRepo(logger *slog.Logger, db *pgxpool.Pool) *StockRepo {
	return &StockRepo{
		logger: logger.With(slog.String("module", "stock-repo")),
		db:     db,
	}
}

func (r *StockRepo) GetBySKUs(ctx context.Context, skus []string) (map[string]*domain.Stock, error) {
	if len(skus) == 0 {
		return map[string]*domain.Stock{}, nil
	}

	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getStocksBySKUs, skus)
	if err != nil {
		r.logger.Error("get stocks by skus", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	stockMap := make(map[string]*domain.Stock, len(skus))

	for rows.Next() {
		var (
			sku           string
			totalQuantity int
		)

		if err := rows.Scan(&sku, &totalQuantity); err != nil {
			r.logger.Error("scan stock row", slog.Any("err", err))
			return nil, err
		}

		stockMap[sku] = domain.StockFromExist(sku, totalQuantity, nil)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	skuList := make([]string, 0, len(stockMap))
	for sku := range stockMap {
		skuList = append(skuList, sku)
	}

	if len(skuList) == 0 {
		return stockMap, nil
	}

	reservations, err := r.getReservationsBySKUs(ctx, skuList)
	if err != nil {
		return nil, err
	}

	for sku, res := range reservations {
		if stock, ok := stockMap[sku]; ok {
			stockMap[sku] = domain.StockFromExist(stock.SKU(), stock.TotalQuantity(), res)
		}
	}

	return stockMap, nil
}

func (r *StockRepo) getReservationsBySKUs(ctx context.Context, skus []string) (map[string][]domain.Reservation, error) {
	querier := GetQuerier(ctx, r.db)

	rows, err := querier.Query(ctx, getReservationsBySKUs, skus)
	if err != nil {
		r.logger.Error("get reservations by skus", slog.Any("err", err))
		return nil, err
	}
	defer rows.Close()

	resMap := make(map[string][]domain.Reservation)

	for rows.Next() {
		var (
			sku string
			res domain.Reservation
		)

		if err := rows.Scan(&sku, &res.OrderID, &res.ReservedQty, &res.BackorderQty); err != nil {
			r.logger.Error("scan reservation row", slog.Any("err", err))
			return nil, err
		}

		resMap[sku] = append(resMap[sku], res)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resMap, nil
}

func (r *StockRepo) Update(ctx context.Context, stock *domain.Stock) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, upsertStock, stock.SKU(), stock.TotalQuantity())
	if err != nil {
		r.logger.Error("upsert stock", slog.Any("err", err), slog.String("sku", stock.SKU()))
		return err
	}

	_, err = querier.Exec(ctx, deleteReservationsBySKU, stock.SKU())
	if err != nil {
		r.logger.Error("delete reservations", slog.Any("err", err), slog.String("sku", stock.SKU()))
		return err
	}

	for _, res := range stock.Reservations() {
		_, err := querier.Exec(ctx, addReservation, res.OrderID, stock.SKU(), res.ReservedQty, res.BackorderQty)
		if err != nil {
			r.logger.Error("add reservation", slog.Any("err", err), slog.String("sku", stock.SKU()))
			return err
		}
	}

	return nil
}
