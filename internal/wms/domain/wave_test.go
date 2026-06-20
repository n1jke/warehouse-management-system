package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestWave_AddOrder(t *testing.T) {
	t.Parallel()

	wave, err := domain.NewWave(2)
	require.NoError(t, err)

	orderID := uuid.New()
	err = wave.AddOrder(orderID)
	require.NoError(t, err)

	assert.Equal(t, 1, len(wave.Orders()))
	assert.Equal(t, orderID, wave.Orders()[0])
}

func TestWave_CloseAndComplete(t *testing.T) {
	t.Parallel()

	wave, err := domain.NewWave(1)
	require.NoError(t, err)

	err = wave.Close()
	require.NoError(t, err)
	assert.Equal(t, domain.WaveStatusInProcess, wave.Status())
	require.NotNil(t, wave.ClosedAt())

	err = wave.Complete()
	require.NoError(t, err)
	assert.Equal(t, domain.WaveStatusCompleted, wave.Status())
}

func TestWave_AddOrderToClosedWave(t *testing.T) {
	t.Parallel()

	wave, err := domain.NewWave(1)
	require.NoError(t, err)

	err = wave.Close()
	require.NoError(t, err)

	err = wave.AddOrder(uuid.New())
	require.Error(t, err)
	assert.Equal(t, domain.ErrWaveNotOpen, err)
}
