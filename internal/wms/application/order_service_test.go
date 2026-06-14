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

func expectTx(t *testing.T, tx *mocks.MockTransactor) {
	tx.EXPECT().WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(t.Context())
		})
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	chatID := int64(1234)

	tests := []struct {
		name    string
		items   []domain.OrderItem
		prepare func(*mocks.MockOrderRepository, *mocks.MockEventPublisher, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name:  "success",
			items: []domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}},
			prepare: func(r *mocks.MockOrderRepository, p *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
				p.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "empty items",
			items: []domain.OrderItem{},
			prepare: func(_ *mocks.MockOrderRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
			},
			wantErr: domain.ErrInvalidOrderItems,
		},
		{
			name:  "add order fail",
			items: []domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}},
			prepare: func(r *mocks.MockOrderRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name:  "publish fail",
			items: []domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}},
			prepare: func(r *mocks.MockOrderRepository, p *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
				p.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name:  "negative quantity",
			items: []domain.OrderItem{{SKU: "LAPTOP-001", Quantity: -1}},
			prepare: func(_ *mocks.MockOrderRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
			},
			wantErr: domain.ErrInvalidItemQuantity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			stockRepo := mocks.NewMockStockRepository(ctrl)
			publisher := mocks.NewMockEventPublisher(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(orderRepo, publisher, tx)

			svc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)

			order, err := svc.CreateOrder(context.Background(), chatID, tt.items)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, order)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, domain.StatusNew, order.Status())
		})
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	orderID := uuid.New()

	tests := []struct {
		name    string
		prepare func(*mocks.MockOrderRepository)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(r *mocks.MockOrderRepository) {
				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}},
					time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
			},
		},
		{
			name: "not found",
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, application.ErrNotFound)
			},
			wantErr: application.ErrOrderNotFound,
		},
		{
			name: "repo error",
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, assert.AnError)
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
			stockRepo := mocks.NewMockStockRepository(ctrl)
			publisher := mocks.NewMockEventPublisher(ctrl)
			tx := mocks.NewMockTransactor(ctrl)

			tt.prepare(orderRepo)

			svc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)

			order, err := svc.GetOrder(context.Background(), orderID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, order)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, orderID, order.ID())
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	t.Parallel()

	orderID := uuid.New()
	newItems := []domain.OrderItem{{SKU: "MONITOR-01", Quantity: 2}}

	tests := []struct {
		name    string
		prepare func(*mocks.MockOrderRepository, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(r *mocks.MockOrderRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				r.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "not found",
			prepare: func(r *mocks.MockOrderRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, application.ErrNotFound)
			},
			wantErr: application.ErrOrderNotFound,
		},
		{
			name: "cannot update shipped",
			prepare: func(r *mocks.MockOrderRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusShipped,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
			},
			wantErr: application.ErrOrderCannotBeUpdated,
		},
		{
			name: "repo error on get",
			prepare: func(r *mocks.MockOrderRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update fail",
			prepare: func(r *mocks.MockOrderRepository, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				r.EXPECT().Update(gomock.Any(), gomock.Any()).Return(assert.AnError)
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
			stockRepo := mocks.NewMockStockRepository(ctrl)
			publisher := mocks.NewMockEventPublisher(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(orderRepo, tx)

			svc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)

			err := svc.UpdateOrder(context.Background(), orderID, newItems)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestListOrders(t *testing.T) {
	t.Parallel()

	now := time.Now()
	order := domain.OrderFromExist(uuid.New(), 999, domain.StatusNew,
		[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, now, now)

	tests := []struct {
		name    string
		input   application.ListOrdersInput
		prepare func(*mocks.MockOrderRepository)
		check   func(*testing.T, application.ListOrdersOutput)
		wantErr error
	}{
		{
			name: "by user id",
			input: application.ListOrdersInput{
				UserID:   999,
				PageSize: 10,
			},
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByUserID(gomock.Any(), int64(999), 11, application.Cursor{}).
					Return([]*domain.Order{order}, nil)
			},
			check: func(t *testing.T, out application.ListOrdersOutput) {
				assert.Len(t, out.Orders, 1)
				assert.Empty(t, out.NextPageToken)
			},
		},
		{
			name: "by status",
			input: application.ListOrdersInput{
				UserID:   999,
				Status:   new("new"),
				PageSize: 10,
			},
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByStatus(gomock.Any(), domain.StatusNew, 11, application.Cursor{}).
					Return([]*domain.Order{order}, nil)
			},
			check: func(t *testing.T, out application.ListOrdersOutput) {
				assert.Len(t, out.Orders, 1)
			},
		},
		{
			name: "pagination next page",
			input: application.ListOrdersInput{
				UserID:   999,
				PageSize: 1,
			},
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByUserID(gomock.Any(), int64(999), 2, application.Cursor{}).
					Return([]*domain.Order{order, order}, nil)
			},
			check: func(t *testing.T, out application.ListOrdersOutput) {
				assert.Len(t, out.Orders, 1)
				assert.NotEmpty(t, out.NextPageToken)
			},
		},
		{
			name: "empty result",
			input: application.ListOrdersInput{
				UserID:   999,
				PageSize: 10,
			},
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByUserID(gomock.Any(), int64(999), 11, application.Cursor{}).
					Return(nil, nil)
			},
			check: func(t *testing.T, out application.ListOrdersOutput) {
				assert.Empty(t, out.Orders)
				assert.Empty(t, out.NextPageToken)
			},
		},
		{
			name: "invalid page token",
			input: application.ListOrdersInput{
				UserID:    999,
				PageToken: "invalid",
				PageSize:  10,
			},
			prepare: func(_ *mocks.MockOrderRepository) {},
			wantErr: application.ErrInvalidPageToken,
		},
		{
			name: "repo error",
			input: application.ListOrdersInput{
				UserID:   999,
				PageSize: 10,
			},
			prepare: func(r *mocks.MockOrderRepository) {
				r.EXPECT().GetByUserID(gomock.Any(), int64(999), 11, application.Cursor{}).
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
			orderRepo := mocks.NewMockOrderRepository(ctrl)
			stockRepo := mocks.NewMockStockRepository(ctrl)
			publisher := mocks.NewMockEventPublisher(ctrl)
			tx := mocks.NewMockTransactor(ctrl)

			tt.prepare(orderRepo)

			svc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)

			out, err := svc.ListOrders(context.Background(), tt.input)
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

func TestDeleteOrder(t *testing.T) {
	t.Parallel()

	orderID := uuid.New()

	tests := []struct {
		name    string
		prepare func(*mocks.MockOrderRepository, *mocks.MockStockRepository, *mocks.MockEventPublisher, *mocks.MockTransactor)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(r *mocks.MockOrderRepository, s *mocks.MockStockRepository, p *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				s.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(map[string]*domain.Stock{}, nil)
				r.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				p.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "not found",
			prepare: func(r *mocks.MockOrderRepository, _ *mocks.MockStockRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, application.ErrNotFound)
			},
			wantErr: application.ErrOrderNotFound,
		},
		{
			name: "cannot cancel shipped",
			prepare: func(r *mocks.MockOrderRepository, _ *mocks.MockStockRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusShipped,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
			},
			wantErr: application.ErrOrderCannotBeCancelled,
		},
		{
			name: "get stocks fail",
			prepare: func(r *mocks.MockOrderRepository, s *mocks.MockStockRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				s.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "stock update fail",
			prepare: func(r *mocks.MockOrderRepository, s *mocks.MockStockRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				s.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(
					map[string]*domain.Stock{"LAPTOP-001": domain.NewStock("LAPTOP-001", 10)}, nil)
				s.EXPECT().Update(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "update order fail",
			prepare: func(r *mocks.MockOrderRepository, s *mocks.MockStockRepository, _ *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				s.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(map[string]*domain.Stock{}, nil)
				r.EXPECT().Update(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
		{
			name: "publish fail",
			prepare: func(r *mocks.MockOrderRepository, s *mocks.MockStockRepository, p *mocks.MockEventPublisher, tx *mocks.MockTransactor) {
				expectTx(t, tx)

				order := domain.OrderFromExist(orderID, 999, domain.StatusNew,
					[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
				r.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
				s.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(map[string]*domain.Stock{}, nil)
				r.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				p.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(assert.AnError)
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
			stockRepo := mocks.NewMockStockRepository(ctrl)
			publisher := mocks.NewMockEventPublisher(ctrl)
			tx := mocks.NewMockTransactor(ctrl)
			tt.prepare(orderRepo, stockRepo, publisher, tx)

			svc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)

			err := svc.DeleteOrder(context.Background(), orderID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
