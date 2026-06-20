package application_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/application/mocks"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestCreateWave(t *testing.T) {
	t.Parallel()

	orderID := uuid.New()
	maxOrders := 5

	tests := []struct {
		name    string
		max     int
		prepare func(*mocks.MockOrderRepository, *mocks.MockWaveRepository, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name: "success",
			max:  maxOrders,
			prepare: func(r *mocks.MockOrderRepository, w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusReserved,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, maxOrders, application.Cursor{}).
					Return([]*domain.Order{order}, nil)
				r.EXPECT().UpdateStatusBatch(gomock.Any(), []uuid.UUID{orderID}, domain.StatusInWave).Return(nil)
				w.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "no orders available",
			max:  maxOrders,
			prepare: func(r *mocks.MockOrderRepository, w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, maxOrders, application.Cursor{}).
					Return([]*domain.Order{}, nil)
				r.EXPECT().UpdateStatusBatch(gomock.Any(), gomock.Any(), domain.StatusInWave).Return(nil)
				w.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "get orders fail",
			max:  maxOrders,
			prepare: func(r *mocks.MockOrderRepository, _ *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, maxOrders, application.Cursor{}).
					Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update batch fail",
			max:  maxOrders,
			prepare: func(r *mocks.MockOrderRepository, _ *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusReserved,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, maxOrders, application.Cursor{}).
					Return([]*domain.Order{order}, nil)
				r.EXPECT().UpdateStatusBatch(gomock.Any(), []uuid.UUID{orderID}, domain.StatusInWave).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "save wave fail",
			max:  maxOrders,
			prepare: func(r *mocks.MockOrderRepository, w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusReserved,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, maxOrders, application.Cursor{}).
					Return([]*domain.Order{order}, nil)
				r.EXPECT().UpdateStatusBatch(gomock.Any(), []uuid.UUID{orderID}, domain.StatusInWave).Return(nil)
				w.EXPECT().Add(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			waveRepo := mocks.NewMockWaveRepository(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(orderRepo, waveRepo, tx)

			svc := application.NewWaveService(logger, tx, waveRepo, orderRepo)

			wave, err := svc.CreateWave(context.Background(), tt.max)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, wave)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetWave(t *testing.T) {
	t.Parallel()

	waveID := uuid.New()

	tests := []struct {
		name    string
		prepare func(*mocks.MockWaveRepository)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(&domain.Wave{}, nil)
			},
		},
		{
			name: "not found",
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(nil, application.ErrNotFound)
			},
			wantErr: application.ErrWaveNotFound,
		},
		{
			name: "repo error",
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			waveRepo := mocks.NewMockWaveRepository(ctrl)
			tt.prepare(waveRepo)

			svc := application.NewWaveService(logger, nil, waveRepo, nil)

			wave, err := svc.GetWave(context.Background(), waveID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, wave)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestListWaves(t *testing.T) {
	t.Parallel()

	waveStatus := domain.WaveStatusOpen

	tests := []struct {
		name    string
		input   application.ListWavesInput
		prepare func(*mocks.MockWaveRepository)
		wantErr error
		check   func(*testing.T, *application.ListWavesOutput)
	}{
		{
			name: "by status",
			input: application.ListWavesInput{
				Status:   &waveStatus,
				PageSize: 10,
			},
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByStatus(gomock.Any(), domain.WaveStatusOpen, 11, application.Cursor{}).
					Return([]*domain.Wave{}, nil)
			},
		},
		{
			name: "default to open",
			input: application.ListWavesInput{
				PageSize: 10,
			},
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByStatus(gomock.Any(), domain.WaveStatusOpen, 11, application.Cursor{}).
					Return([]*domain.Wave{}, nil)
			},
		},
		{
			name: "pagination next page",
			input: application.ListWavesInput{
				PageSize: 2,
			},
			prepare: func(w *mocks.MockWaveRepository) {
				waves := make([]*domain.Wave, 3)
				for i := range waves {
					waves[i] = domain.WaveFromExist(
						uuid.New(),
						domain.WaveStatusOpen,
						nil, 0,
						time.Date(2024, 1, 1+i, 0, 0, 0, 0, time.UTC),
						nil,
					)
				}

				w.EXPECT().GetByStatus(gomock.Any(), domain.WaveStatusOpen, 3, application.Cursor{}).
					Return(waves, nil)
			},
			check: func(t *testing.T, out *application.ListWavesOutput) {
				require.Len(t, out.Waves, 2)
				require.NotEmpty(t, out.NextPageToken)
			},
		},
		{
			name: "invalid page token",
			input: application.ListWavesInput{
				PageToken: "bad-token",
				PageSize:  10,
			},
			prepare: func(_ *mocks.MockWaveRepository) {},
			wantErr: application.ErrInvalidPageToken,
		},
		{
			name: "empty result",
			input: application.ListWavesInput{
				PageSize: 10,
			},
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByStatus(gomock.Any(), domain.WaveStatusOpen, 11, application.Cursor{}).
					Return([]*domain.Wave{}, nil)
			},
			check: func(t *testing.T, out *application.ListWavesOutput) {
				require.Empty(t, out.Waves)
				require.Empty(t, out.NextPageToken)
			},
		},
		{
			name: "repo error",
			input: application.ListWavesInput{
				PageSize: 10,
			},
			prepare: func(w *mocks.MockWaveRepository) {
				w.EXPECT().GetByStatus(gomock.Any(), domain.WaveStatusOpen, 11, application.Cursor{}).
					Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			waveRepo := mocks.NewMockWaveRepository(ctrl)
			tt.prepare(waveRepo)

			svc := application.NewWaveService(logger, nil, waveRepo, nil)

			out, err := svc.ListWaves(context.Background(), tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, out)
			}
		})
	}
}

func TestCloseWave(t *testing.T) {
	t.Parallel()

	waveID := uuid.New()

	tests := []struct {
		name    string
		prepare func(*mocks.MockWaveRepository, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusOpen, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
				w.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "get wave fail",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update fail",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusOpen, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
				w.EXPECT().Update(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "already closed",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusInProcess, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
			},
			wantErr: domain.ErrWaveNotOpen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			waveRepo := mocks.NewMockWaveRepository(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(waveRepo, tx)

			svc := application.NewWaveService(logger, tx, waveRepo, nil)

			err := svc.CloseWave(context.Background(), waveID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCompleteWave(t *testing.T) {
	t.Parallel()

	waveID := uuid.New()

	tests := []struct {
		name    string
		prepare func(*mocks.MockWaveRepository, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusInProcess, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
				w.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "get wave fail",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update fail",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusInProcess, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
				w.EXPECT().Update(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "not in process",
			prepare: func(w *mocks.MockWaveRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				wave := domain.WaveFromExist(waveID, domain.WaveStatusOpen, nil, 10, time.Now(), nil)
				w.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
			},
			wantErr: domain.ErrWaveNotInProcess,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			waveRepo := mocks.NewMockWaveRepository(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(waveRepo, tx)

			svc := application.NewWaveService(logger, tx, waveRepo, nil)

			err := svc.CompleteWave(context.Background(), waveID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
