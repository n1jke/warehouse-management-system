package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type OrderEventType string

const (
	EventOrderCreated           OrderEventType = "order-created"
	EventOrderCancelled         OrderEventType = "order-cancelled"
	EventOrderReserved          OrderEventType = "order-reserved"
	EventOrderPartiallyReserved OrderEventType = "order-partially_reserved"
	EventOrderInWave            OrderEventType = "order-in_wave"
	EventOrderShipped           OrderEventType = "order-shipped"
)

type OrderEvent struct {
	EventID    uuid.UUID
	EventType  OrderEventType
	OrderID    uuid.UUID
	UserID     int64
	Status     domain.OrderStatus
	OccurredAt time.Time
}

func NewOrderEvent(event OrderEventType, order *domain.Order) (*OrderEvent, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate uuid v7: %w", err)
	}

	return &OrderEvent{
		EventID:    id,
		EventType:  event,
		OrderID:    order.ID(),
		UserID:     order.UserID(),
		Status:     order.Status(),
		OccurredAt: time.Now(),
	}, nil
}
