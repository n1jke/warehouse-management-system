package domain

import "github.com/google/uuid"

type Reservation struct {
	OrderID      uuid.UUID
	ReservedQty  int
	BackorderQty int
}

func DetermineOrderStatus(reservations []Reservation) OrderStatus {
	for _, r := range reservations {
		if r.BackorderQty > 0 {
			return StatusPartiallyReserved
		}
	}

	return StatusReserved
}
