package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFullWave    = errors.New("wave is full")
	ErrWaveNotOpen = errors.New("cannot add orders to non-open wave")
)

type WaveStatus string

const (
	WaveStatusOpen      WaveStatus = "open"
	WaveStatusInProcess WaveStatus = "in_process"
	WaveStatusCompleted WaveStatus = "completed"
)

type Wave struct {
	id        uuid.UUID
	orders    []int64
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
		orders:    make([]int64, 0),
		createdAt: time.Now(),
	}, nil
}

func (w *Wave) AddOrder(orderID int64) error {
	if w.status != WaveStatusOpen {
		return ErrWaveNotOpen
	}

	if w.IsFull() {
		return ErrFullWave
	}

	w.orders = append(w.orders, orderID)

	return nil
}

func (w *Wave) Orders() []int64 { return slices.Clone(w.orders) }

func (w *Wave) IsFull() bool { return len(w.orders) >= w.maxOrders }

func (w *Wave) Close() {
	t := time.Now()
	w.closedAt = &t
}
