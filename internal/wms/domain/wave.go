package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFullWave         = errors.New("wave is full")
	ErrWaveNotOpen      = errors.New("cannot add orders to non-open wave")
	ErrWaveNotInProcess = errors.New("only in-process wave can be completed")
)

type WaveStatus string

const (
	WaveStatusOpen      WaveStatus = "open"
	WaveStatusInProcess WaveStatus = "in_process"
	WaveStatusCompleted WaveStatus = "completed"
)

type Wave struct {
	id        uuid.UUID
	orders    []uuid.UUID
	status    WaveStatus
	maxOrders int
	createdAt time.Time
	closedAt  *time.Time
}

func NewWave(maxOrders int) (*Wave, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate uuid v7: %w", err)
	}

	return &Wave{
		id:        id,
		status:    WaveStatusOpen,
		maxOrders: maxOrders,
		orders:    make([]uuid.UUID, 0),
		createdAt: time.Now(),
	}, nil
}

func WaveFromExist(id uuid.UUID, status WaveStatus, orders []uuid.UUID, maxOrders int, createdAt time.Time, closedAt *time.Time) *Wave {
	return &Wave{
		id:        id,
		status:    status,
		orders:    orders,
		maxOrders: maxOrders,
		createdAt: createdAt,
		closedAt:  closedAt,
	}
}

func (w *Wave) AddOrder(orderID uuid.UUID) error {
	if w.status != WaveStatusOpen {
		return ErrWaveNotOpen
	}

	if w.IsFull() {
		return ErrFullWave
	}

	w.orders = append(w.orders, orderID)

	return nil
}

func (w *Wave) ID() uuid.UUID { return w.id }

func (w *Wave) Status() WaveStatus { return w.status }

func (w *Wave) CreatedAt() time.Time { return w.createdAt }

func (w *Wave) ClosedAt() *time.Time { return w.closedAt }

func (w *Wave) Orders() []uuid.UUID { return slices.Clone(w.orders) }

func (w *Wave) IsFull() bool { return len(w.orders) >= w.maxOrders }

func (w *Wave) Close() error {
	if w.status != WaveStatusOpen {
		return ErrWaveNotOpen
	}

	t := time.Now()
	w.closedAt = &t
	w.status = WaveStatusInProcess

	return nil
}

func (w *Wave) Complete() error {
	if w.status != WaveStatusInProcess {
		return ErrWaveNotInProcess
	}

	w.status = WaveStatusCompleted

	return nil
}
