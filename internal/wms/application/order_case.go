package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type ListOrdersInput struct {
	UserID    int64
	Status    *string
	PageToken string
	PageSize  int
}

type ListOrdersOutput struct {
	Orders        []*domain.Order
	NextPageToken string
}

type OrderService struct {
	logger    *slog.Logger
	tx        Transactor
	orderRepo OrderRepository
	stockRepo StockRepository
	publisher EventPublisher
}

func NewOrderService(logger *slog.Logger, tx Transactor, orderRepo OrderRepository, stockRepo StockRepository, publisher EventPublisher,
) *OrderService {
	return &OrderService{
		logger:    logger.With(slog.String("module", "order-service")),
		tx:        tx,
		orderRepo: orderRepo,
		stockRepo: stockRepo,
		publisher: publisher,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, chatID int64, items []domain.OrderItem) error {
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.createOrderTx(ctx, chatID, items)
	})
	if err != nil {
		s.logger.Error("create order tx", slog.Int64("chatID", chatID), slog.Any("err", err))
		return fmt.Errorf("create order tx: %w", err)
	}

	return nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrOrderNotFound
		}

		s.logger.Error("get order by id", slog.Any("orderID", orderID), slog.Any("err", err))

		return nil, fmt.Errorf("get order by id: %w", err)
	}

	return order, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.updateOrderTx(ctx, orderID, items)
	})
	if err != nil {
		s.logger.Error("update order tx", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("update order tx: %w", err)
	}

	return nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, orderID uuid.UUID) error {
	err := s.tx.WithTransaction(ctx, func(ctx context.Context) error {
		return s.deleteOrderTx(ctx, orderID)
	})
	if err != nil {
		s.logger.Error("delete order tx", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("delete order tx: %w", err)
	}

	return nil
}

func (s *OrderService) ListOrders(ctx context.Context, input ListOrdersInput) (ListOrdersOutput, error) {
	s.logger.Info("list orders start", slog.Int64("user_id", input.UserID),
		slog.Any("status", input.Status), slog.Int("page_size", input.PageSize))

	cursor, err := decodeCursor(input.PageToken)
	if err != nil {
		return ListOrdersOutput{}, fmt.Errorf("decode cursor: %w", err)
	}

	limit := input.PageSize

	orders, err := s.fetchOrders(ctx, input, limit+1, cursor)
	if err != nil {
		s.logger.Error("list orders fetch", slog.Any("err", err))
		return ListOrdersOutput{}, fmt.Errorf("list orders: %w", err)
	}

	nextPageToken := ""

	if len(orders) == limit+1 {
		orders = orders[:limit]

		last := orders[len(orders)-1]

		token, err := encodeCursor(Cursor{
			LastCreatedAt: last.CreatedAt(),
			LastID:        last.ID(),
		})
		if err != nil {
			return ListOrdersOutput{}, fmt.Errorf("encode cursor: %w", err)
		}

		nextPageToken = token
	}

	s.logger.Info("list orders end", slog.Int("count", len(orders)))

	return ListOrdersOutput{
		Orders:        orders,
		NextPageToken: nextPageToken,
	}, nil
}
