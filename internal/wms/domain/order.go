package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidOrderItems   = errors.New("order must have at least one item")
	ErrInvalidUpdateTime   = errors.New("new update time before current")
	ErrInvalidItemQuantity = errors.New("item quantity must be positive")
)

type OrderStatus string

const (
	StatusNew               OrderStatus = "new"
	StatusReserving         OrderStatus = "reserving"
	StatusReserved          OrderStatus = "reserved"
	StatusPartiallyReserved OrderStatus = "partially_reserved"
	StatusInWave            OrderStatus = "in_wave"
	StatusShipped           OrderStatus = "shipped"
	StatusCancelled         OrderStatus = "cancelled"
)

type OrderItem struct {
	SKU      string
	Quantity int
}

type Order struct {
	id        uuid.UUID
	userID    int64
	status    OrderStatus
	items     []OrderItem
	createdAt time.Time
	updatedAt time.Time
}

func NewOrder(userID int64, items []OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrInvalidOrderItems
	}

	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, ErrInvalidItemQuantity
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate uuid v7: %w", err)
	}

	return &Order{
		id:        id,
		userID:    userID,
		status:    StatusNew,
		items:     items,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}, nil
}

func OrderFromExist(id uuid.UUID, userID int64, status OrderStatus, items []OrderItem, createdAt, updatedAt time.Time) *Order {
	return &Order{
		id:        id,
		userID:    userID,
		status:    status,
		items:     items,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (o *Order) ID() uuid.UUID { return o.id }

func (o *Order) UserID() int64 { return o.userID }

func (o *Order) Status() OrderStatus { return o.status }

func (o *Order) Items() []OrderItem { return slices.Clone(o.items) }

func (o *Order) CreatedAt() time.Time { return o.createdAt }

func (o *Order) UpdatedAt() time.Time { return o.updatedAt }

func (o *Order) TransitionTo(next OrderStatus) error {
	if !defaultFSM.IsAllowed(o.status, next) {
		return NewErrOrderStatusFSM(o.status, next)
	}

	o.status = next
	o.updatedAt = time.Now()

	return nil
}

func (o *Order) CanUpdate() bool { return o.status == StatusNew || o.status == StatusReserving }

func (o *Order) CanCancel() bool { return o.status != StatusShipped && o.status != StatusCancelled }
