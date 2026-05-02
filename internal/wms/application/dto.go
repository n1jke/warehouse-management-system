package application

import (
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
