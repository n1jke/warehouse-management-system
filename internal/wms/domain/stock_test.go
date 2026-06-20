package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestStock_Reserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		total        int
		requested    int
		wantReserved int
		wantBacklog  int
	}{
		{
			name:         "full reserve",
			total:        10,
			requested:    5,
			wantReserved: 5,
			wantBacklog:  0,
		},
		{
			name:         "partial reserve with backorder",
			total:        3,
			requested:    5,
			wantReserved: 3,
			wantBacklog:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stock := domain.NewStock("sku-1", tt.total)
			orderID := uuid.New()
			res := stock.Reserve(orderID, tt.requested)

			assert.Equal(t, tt.wantReserved, res.ReservedQty)
			assert.Equal(t, tt.wantBacklog, res.BackorderQty)

			if tt.wantReserved > 0 {
				assert.Equal(t, tt.total-tt.wantReserved, stock.Available())
			}
		})
	}
}

func TestStock_Release(t *testing.T) {
	t.Parallel()

	stock := domain.NewStock("sku-1", 5)
	orderID := uuid.New()
	stock.Reserve(orderID, 3)

	stock.Release(orderID)

	assert.Equal(t, 5, stock.Available())
}
