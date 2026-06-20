package domain

import (
	"slices"

	"github.com/google/uuid"
)

type Stock struct {
	sku           string
	totalQuantity int
	reservations  []Reservation
}

func NewStock(sku string, total int) *Stock {
	return &Stock{
		sku:           sku,
		totalQuantity: total,
	}
}

func StockFromExist(sku string, total int, reservations []Reservation) *Stock {
	return &Stock{
		sku:           sku,
		totalQuantity: total,
		reservations:  reservations,
	}
}

func (s *Stock) SKU() string { return s.sku }

func (s *Stock) TotalQuantity() int { return s.totalQuantity }

func (s *Stock) Reservations() []Reservation { return slices.Clone(s.reservations) }

func (s *Stock) Available() int {
	reserved := 0
	for _, r := range s.reservations {
		reserved += r.ReservedQty
	}

	return max(s.totalQuantity-reserved, 0)
}

func (s *Stock) Reserve(orderID uuid.UUID, requestedQty int) Reservation {
	toReserve := min(requestedQty, s.Available())

	r := Reservation{
		OrderID:      orderID,
		SKU:          s.sku,
		ReservedQty:  toReserve,
		BackorderQty: requestedQty - toReserve,
	}

	if toReserve > 0 {
		s.reservations = append(s.reservations, r)
	}

	return r
}

func (s *Stock) Release(orderID uuid.UUID) {
	s.reservations = slices.DeleteFunc(s.reservations, func(r Reservation) bool {
		return r.OrderID == orderID
	})
}
