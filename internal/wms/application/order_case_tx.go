package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func (s *OrderService) createOrderTx(ctx context.Context, chatID int64, items []domain.OrderItem) (*domain.Order, error) {
	order, err := domain.NewOrder(chatID, items)
	if err != nil {
		s.logger.Error("create order", slog.Int64("chatID", chatID), slog.Any("err", err))
		return nil, fmt.Errorf("create order: %w", err)
	}

	if err := s.orderRepo.Add(ctx, order); err != nil {
		s.logger.Error("add order to repo", slog.Int64("chatID", chatID), slog.Any("err", err))
		return nil, fmt.Errorf("add order to repo: %w", err)
	}

	event, err := NewOrderEvent(EventOrderCreated, order)
	if err != nil {
		s.logger.Error("create order event", slog.Int64("chatID", chatID), slog.Any("err", err))
		return nil, fmt.Errorf("create order event: %w", err)
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		s.logger.Error("publish order created event", slog.Int64("chatID", chatID), slog.Any("err", err))
		return nil, fmt.Errorf("publish order created event: %w", err)
	}

	return order, nil
}

func (s *OrderService) updateOrderTx(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrOrderNotFound
		}

		s.logger.Error("get order by id", slog.Any("orderID", orderID), slog.Any("err", err))

		return fmt.Errorf("get order by id: %w", err)
	}

	if !order.CanUpdate() {
		return ErrOrderCannotBeUpdated
	}

	order.UpdateItems(items)

	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("update order in repo", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("update order in repo: %w", err)
	}

	return nil
}

func (s *OrderService) deleteOrderTx(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrOrderNotFound
		}

		s.logger.Error("get order by id", slog.Any("orderID", orderID), slog.Any("err", err))

		return fmt.Errorf("get order by id: %w", err)
	}

	if !order.CanCancel() {
		return ErrOrderCannotBeCancelled
	}

	// todo: update StockRepository

	if err := order.TransitionTo(domain.StatusCancelled); err != nil {
		s.logger.Error("transition order to cancelled", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("transition order to cancelled: %w", err)
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("update cancelled order", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("update cancelled order: %w", err)
	}

	event, err := NewOrderEvent(EventOrderCancelled, order)
	if err != nil {
		s.logger.Error("create order event", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("create order event: %w", err)
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		s.logger.Error("publish order cancelled event", slog.Any("orderID", orderID), slog.Any("err", err))
		return fmt.Errorf("publish order cancelled event: %w", err)
	}

	return nil
}

func (s *OrderService) fetchOrders(ctx context.Context, input ListOrdersInput, limit int, cursor Cursor) ([]*domain.Order, error) {
	if input.Status != nil {
		status := domain.OrderStatus(*input.Status)
		return s.orderRepo.GetByStatus(ctx, status, limit, cursor)
	}

	return s.orderRepo.GetByUserID(ctx, input.UserID, limit, cursor)
}
