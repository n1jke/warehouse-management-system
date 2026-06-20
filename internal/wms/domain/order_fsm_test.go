package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestOrderFSM_IsAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from domain.OrderStatus
		to   domain.OrderStatus
		want bool
	}{
		{
			name: "new to reserving",
			from: domain.StatusNew,
			to:   domain.StatusReserving,
			want: true,
		},
		{
			name: "new to shipped not allowed",
			from: domain.StatusNew,
			to:   domain.StatusShipped,
			want: false,
		},
		{
			name: "reserved to in_wave",
			from: domain.StatusReserved,
			to:   domain.StatusInWave,
			want: true,
		},
		{
			name: "shipped to cancelled not allowed",
			from: domain.StatusShipped,
			to:   domain.StatusCancelled,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := domain.NewOrderFSM().IsAllowed(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}
