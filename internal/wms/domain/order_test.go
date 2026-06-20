package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestNewOrder_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []domain.OrderItem
	}{
		{
			name:  "empty items",
			items: nil,
		},
		{
			name:  "negative quantity",
			items: []domain.OrderItem{{SKU: "sku-1", Quantity: -1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order, err := domain.NewOrder(1, tt.items)
			require.Error(t, err)
			assert.Nil(t, order)
		})
	}
}

func TestNewOrder_Success(t *testing.T) {
	t.Parallel()

	items := []domain.OrderItem{
		{SKU: "sku-1", Quantity: 5},
		{SKU: "sku-2", Quantity: 2},
	}

	order, err := domain.NewOrder(42, items)
	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, int64(42), order.UserID())
	assert.Equal(t, domain.StatusNew, order.Status())
	assert.Equal(t, items, order.Items())
	assert.NotZero(t, order.ID())
}

func TestOrder_TransitionTo(t *testing.T) {
	t.Parallel()

	order, err := domain.NewOrder(10, []domain.OrderItem{{SKU: "sku-1", Quantity: 1}})
	require.NoError(t, err)

	err = order.TransitionTo(domain.StatusReserving)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusReserving, order.Status())
}
