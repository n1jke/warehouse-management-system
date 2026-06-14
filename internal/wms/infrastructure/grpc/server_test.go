package grpc_test

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/n1jke/warehouse-management-system/internal/api/proto/wms"
	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/application/mocks"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
	grpcserver "github.com/n1jke/warehouse-management-system/internal/wms/infrastructure/grpc"
)

func expectTx(t *testing.T, tx *mocks.MockTransactor) {
	tx.EXPECT().WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
			return fn(t.Context())
		})
}

func assertCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()

	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %v", err)
	assert.Equal(t, expected, st.Code())
}

//nolint:gocritic // only test usage
func newServer(ctrl *gomock.Controller) (*grpcserver.Server, *mocks.MockOrderRepository, *mocks.MockStockRepository,
	*mocks.MockEventPublisher, *mocks.MockTransactor, *mocks.MockUserRepository, *mocks.MockWaveRepository,
) {
	orderRepo := mocks.NewMockOrderRepository(ctrl)
	stockRepo := mocks.NewMockStockRepository(ctrl)
	publisher := mocks.NewMockEventPublisher(ctrl)
	tx := mocks.NewMockTransactor(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	waveRepo := mocks.NewMockWaveRepository(ctrl)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	userSvc := application.NewUserService(logger, userRepo)
	orderSvc := application.NewOrderService(logger, tx, orderRepo, stockRepo, publisher)
	waveSvc := application.NewWaveService(logger, tx, waveRepo, orderRepo)

	return grpcserver.NewServer(userSvc, orderSvc, waveSvc), orderRepo, stockRepo, publisher, tx, userRepo, waveRepo
}

func TestServer_RegisterUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

		_, err := srv.RegisterUser(context.Background(), &wms.RegisterUserRequest{ChatId: 999})
		require.NoError(t, err)
	})

	t.Run("already exists", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().Add(gomock.Any(), gomock.Any()).Return(application.ErrAlreadyExists)

		_, err := srv.RegisterUser(context.Background(), &wms.RegisterUserRequest{ChatId: 999})
		assertCode(t, err, codes.AlreadyExists)
	})

	t.Run("repo error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().Add(gomock.Any(), gomock.Any()).Return(assert.AnError)

		_, err := srv.RegisterUser(context.Background(), &wms.RegisterUserRequest{ChatId: 999})
		assertCode(t, err, codes.Internal)
	})
}

func TestServer_GetUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().GetByChatID(gomock.Any(), int64(1)).Return(domain.NewUser(1), nil)

		resp, err := srv.GetUser(context.Background(), &wms.GetUserRequest{ChatId: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resp.GetChatId())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().GetByChatID(gomock.Any(), int64(1)).Return(nil, application.ErrNotFound)

		_, err := srv.GetUser(context.Background(), &wms.GetUserRequest{ChatId: 1})
		assertCode(t, err, codes.NotFound)
	})

	t.Run("repo error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, userRepo, _ := newServer(ctrl)
		userRepo.EXPECT().GetByChatID(gomock.Any(), int64(1)).Return(nil, assert.AnError)

		_, err := srv.GetUser(context.Background(), &wms.GetUserRequest{ChatId: 1})
		assertCode(t, err, codes.Internal)
	})
}

func TestServer_CreateOrder(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, publisher, tx, _, _ := newServer(ctrl)
		expectTx(t, tx)
		orderRepo.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
		publisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

		resp, err := srv.CreateOrder(context.Background(), &wms.CreateOrderRequest{
			ChatId: 1, Items: []*wms.OrderItem{{Sku: "LAPTOP-001", Quantity: 1}},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetOrderId())
	})

	t.Run("empty items", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, tx, _, _ := newServer(ctrl)
		expectTx(t, tx)

		_, err := srv.CreateOrder(context.Background(), &wms.CreateOrderRequest{
			ChatId: 1, Items: []*wms.OrderItem{},
		})
		assertCode(t, err, codes.Internal)
	})
}

func TestServer_GetOrder(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, _, _, _ := newServer(ctrl)
		orderID := uuid.New()
		order := domain.OrderFromExist(orderID, 1, domain.StatusNew,
			[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
		resp, err := srv.GetOrder(context.Background(), &wms.GetOrderRequest{OrderId: orderID.String()})
		require.NoError(t, err)
		assert.Equal(t, orderID.String(), resp.GetOrderId())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, _, _, _ := newServer(ctrl)
		orderID := uuid.New()
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(nil, application.ErrNotFound)
		_, err := srv.GetOrder(context.Background(), &wms.GetOrderRequest{OrderId: orderID.String()})
		assertCode(t, err, codes.NotFound)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, _ := newServer(ctrl)
		_, err := srv.GetOrder(context.Background(), &wms.GetOrderRequest{OrderId: "bad-uuid"})
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestServer_UpdateOrder(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, tx, _, _ := newServer(ctrl)
		orderID := uuid.New()
		order := domain.OrderFromExist(orderID, 1, domain.StatusNew,
			[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())

		expectTx(t, tx)
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
		orderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
		_, err := srv.UpdateOrder(context.Background(), &wms.UpdateOrderRequest{
			OrderId: orderID.String(),
			Items:   []*wms.OrderItem{{Sku: "LAPTOP-001", Quantity: 2}},
		})
		require.NoError(t, err)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, _ := newServer(ctrl)
		_, err := srv.UpdateOrder(context.Background(), &wms.UpdateOrderRequest{
			OrderId: "bad-uuid",
			Items:   []*wms.OrderItem{{Sku: "LAPTOP-001", Quantity: 2}},
		})
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestServer_DeleteOrder(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, stockRepo, publisher, tx, _, _ := newServer(ctrl)
		orderID := uuid.New()
		order := domain.OrderFromExist(orderID, 1, domain.StatusNew,
			[]domain.OrderItem{{SKU: "LAPTOP-001", Quantity: 1}}, time.Now(), time.Now())

		expectTx(t, tx)
		orderRepo.EXPECT().GetByID(gomock.Any(), orderID).Return(order, nil)
		stockRepo.EXPECT().GetBySKUs(gomock.Any(), []string{"LAPTOP-001"}).Return(map[string]*domain.Stock{}, nil)
		orderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		publisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

		resp, err := srv.DeleteOrder(context.Background(), &wms.DeleteOrderRequest{OrderId: orderID.String()})
		require.NoError(t, err)
		assert.Equal(t, orderID.String(), resp.GetOrderId())
	})

	t.Run("invalid uuid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, _ := newServer(ctrl)
		_, err := srv.DeleteOrder(context.Background(), &wms.DeleteOrderRequest{OrderId: "bad-uuid"})
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestServer_ListOrders(t *testing.T) {
	t.Parallel()

	t.Run("by user", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, _, _, _ := newServer(ctrl)
		orderRepo.EXPECT().GetByUserID(gomock.Any(), int64(1), 11, application.Cursor{}).
			Return([]*domain.Order{}, nil)

		resp, err := srv.ListOrders(context.Background(), &wms.ListOrdersRequest{ChatId: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Empty(t, resp.GetOrders())
	})

	t.Run("by status", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, _, _, _ := newServer(ctrl)
		orderRepo.EXPECT().GetByStatus(gomock.Any(), domain.OrderStatus("new"), 11, application.Cursor{}).
			Return([]*domain.Order{}, nil)

		resp, err := srv.ListOrders(context.Background(), &wms.ListOrdersRequest{
			ChatId: 1, PageSize: 10, Status: "new",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetOrders())
	})

	t.Run("invalid page token", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, _ := newServer(ctrl)
		_, err := srv.ListOrders(context.Background(), &wms.ListOrdersRequest{
			ChatId: 1, PageSize: 10, PageToken: "bad-token",
		})
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestServer_GetWave(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, waveRepo := newServer(ctrl)
		waveID := uuid.New()
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(domain.WaveFromExist(waveID, domain.WaveStatusOpen, nil, 10, time.Now(), nil), nil)
		resp, err := srv.GetWave(context.Background(), &wms.GetWaveRequest{WaveId: waveID.String()})
		require.NoError(t, err)
		assert.Equal(t, waveID.String(), resp.GetWaveId())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, waveRepo := newServer(ctrl)
		waveID := uuid.New()
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(nil, application.ErrNotFound)
		_, err := srv.GetWave(context.Background(), &wms.GetWaveRequest{WaveId: waveID.String()})
		assertCode(t, err, codes.NotFound)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, _, _, _ := newServer(ctrl)
		_, err := srv.GetWave(context.Background(), &wms.GetWaveRequest{WaveId: "bad-uuid"})
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestServer_CreateWave(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, orderRepo, _, _, tx, _, waveRepo := newServer(ctrl)
		expectTx(t, tx)
		orderRepo.EXPECT().GetByStatus(gomock.Any(), domain.StatusReserved, 5, application.Cursor{}).
			Return([]*domain.Order{}, nil)
		orderRepo.EXPECT().UpdateStatusBatch(gomock.Any(), gomock.Any(), domain.StatusInWave).Return(nil)
		waveRepo.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

		resp, err := srv.CreateWave(context.Background(), &wms.CreateWaveRequest{MaxOrders: 5})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetWaveId())
	})
}

func TestServer_CloseWave(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, tx, _, waveRepo := newServer(ctrl)
		waveID := uuid.New()
		wave := domain.WaveFromExist(waveID, domain.WaveStatusOpen, nil, 10, time.Now(), nil)

		expectTx(t, tx)
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
		waveRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
		resp, err := srv.CloseWave(context.Background(), &wms.CloseWaveRequest{WaveId: waveID.String()})
		require.NoError(t, err)
		assert.Equal(t, waveID.String(), resp.GetWaveId())
	})
}

func TestServer_CompleteWave(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		srv, _, _, _, tx, _, waveRepo := newServer(ctrl)
		waveID := uuid.New()
		wave := domain.WaveFromExist(waveID, domain.WaveStatusInProcess, nil, 10, time.Now(), nil)

		expectTx(t, tx)
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
		waveRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		waveRepo.EXPECT().GetByID(gomock.Any(), waveID).Return(wave, nil)
		resp, err := srv.CompleteWave(context.Background(), &wms.CompleteWaveRequest{WaveId: waveID.String()})
		require.NoError(t, err)
		assert.Equal(t, waveID.String(), resp.GetWaveId())
	})
}
