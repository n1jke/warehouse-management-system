package domain

import (
	"errors"
	"fmt"
	"slices"
)

var defaultFSM = NewOrderFSM()

type OrderFSM struct {
	transitions map[OrderStatus][]OrderStatus
}

func NewOrderFSM() *OrderFSM {
	fsm := &OrderFSM{
		transitions: map[OrderStatus][]OrderStatus{
			StatusNew:               {StatusReserving, StatusCancelled},
			StatusReserving:         {StatusReserved, StatusPartiallyReserved, StatusCancelled},
			StatusReserved:          {StatusInWave, StatusCancelled},
			StatusPartiallyReserved: {StatusInWave, StatusCancelled},
			StatusInWave:            {StatusShipped},
			StatusShipped:           {},
			StatusCancelled:         {},
		},
	}

	return fsm
}

func (o *OrderFSM) IsAllowed(from, to OrderStatus) bool {
	allowed, ok := o.transitions[from]
	if !ok {
		return false
	}

	return slices.Contains(allowed, to)
}

var ErrInvalidFSMTransition = errors.New("invalid transition")

type ErrOrderStatusFSM struct {
	From OrderStatus
	To   OrderStatus
	Err  error
}

func NewErrOrderStatusFSM(from, to OrderStatus) error {
	return ErrOrderStatusFSM{
		From: from,
		To:   to,
		Err:  ErrInvalidFSMTransition,
	}
}

func (e ErrOrderStatusFSM) Error() string { return fmt.Sprintf("%s -> %s : %v", e.From, e.To, e.Err) }

func (e ErrOrderStatusFSM) Unwrap() error { return e.Err }
