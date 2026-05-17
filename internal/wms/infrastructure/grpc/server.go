package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/n1jke/warehouse-management-system/internal/api/proto/wms"
	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type Server struct {
	wms.UnimplementedUserServiceServer
	wms.UnimplementedOrderServiceServer
	wms.UnimplementedWaveServiceServer

	userService  *application.UserService
	orderService *application.OrderService
	waveService  *application.WaveService
}

func NewServer(userSvc *application.UserService, orderSvc *application.OrderService, waveSvc *application.WaveService) *Server {
	return &Server{
		userService:  userSvc,
		orderService: orderSvc,
		waveService:  waveSvc,
	}
}

func (s *Server) RegisterUser(ctx context.Context, req *wms.RegisterUserRequest) (*wms.UserResponse, error) {
	user, err := s.userService.RegisterUser(ctx, req.GetChatId())
	if err != nil {
		return nil, mapErrors(err)
	}

	return &wms.UserResponse{Id: user.ID(), ChatId: user.ID()}, nil
}

func (s *Server) GetUser(ctx context.Context, req *wms.GetUserRequest) (*wms.UserResponse, error) {
	user, err := s.userService.GetUser(ctx, req.GetChatId())
	if err != nil {
		return nil, mapErrors(err)
	}

	return &wms.UserResponse{Id: user.ID(), ChatId: user.ID()}, nil
}

func (s *Server) CreateOrder(ctx context.Context, req *wms.CreateOrderRequest) (*wms.OrderResponse, error) {
	items := itemsFromProto(req.GetItems())

	order, err := s.orderService.CreateOrder(ctx, req.GetChatId(), items)
	if err != nil {
		return nil, mapErrors(err)
	}

	return orderToProto(order), nil
}

func (s *Server) GetOrder(ctx context.Context, req *wms.GetOrderRequest) (*wms.OrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	order, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return nil, mapErrors(err)
	}

	return orderToProto(order), nil
}

func (s *Server) UpdateOrder(ctx context.Context, req *wms.UpdateOrderRequest) (*wms.OrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	items := itemsFromProto(req.GetItems())
	if err := s.orderService.UpdateOrder(ctx, orderID, items); err != nil {
		return nil, mapErrors(err)
	}

	order, err := s.orderService.GetOrder(ctx, orderID)
	if err != nil {
		return nil, mapErrors(err)
	}

	return orderToProto(order), nil
}

func (s *Server) DeleteOrder(ctx context.Context, req *wms.DeleteOrderRequest) (*wms.DeleteOrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	if err := s.orderService.DeleteOrder(ctx, orderID); err != nil {
		return nil, mapErrors(err)
	}

	return &wms.DeleteOrderResponse{OrderId: req.GetOrderId()}, nil
}

func (s *Server) ListOrders(ctx context.Context, req *wms.ListOrdersRequest) (*wms.ListOrdersResponse, error) {
	input := application.ListOrdersInput{
		UserID:    req.GetChatId(),
		PageToken: req.GetPageToken(),
		PageSize:  int(req.GetPageSize()),
	}
	if statusStr := req.GetStatus(); statusStr != "" {
		input.Status = &statusStr
	}

	output, err := s.orderService.ListOrders(ctx, input)
	if err != nil {
		return nil, mapErrors(err)
	}

	resp := &wms.ListOrdersResponse{NextPageToken: output.NextPageToken}
	for _, order := range output.Orders {
		resp.Orders = append(resp.Orders, orderToProto(order))
	}

	return resp, nil
}

func (s *Server) GetWave(ctx context.Context, req *wms.GetWaveRequest) (*wms.WaveResponse, error) {
	waveID, err := uuid.Parse(req.GetWaveId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid wave_id")
	}

	wave, err := s.waveService.GetWave(ctx, waveID)
	if err != nil {
		return nil, mapErrors(err)
	}

	return waveToProto(wave), nil
}

func (s *Server) ListWaves(ctx context.Context, req *wms.ListWavesRequest) (*wms.ListWavesResponse, error) {
	input := application.ListWavesInput{
		PageToken: req.GetPageToken(),
		PageSize:  int(req.GetPageSize()),
	}
	if statusStr := req.GetStatus(); statusStr != "" {
		ws := domain.WaveStatus(statusStr)
		input.Status = &ws
	}

	output, err := s.waveService.ListWaves(ctx, input)
	if err != nil {
		return nil, mapErrors(err)
	}

	resp := &wms.ListWavesResponse{NextPageToken: output.NextPageToken}
	for i := range output.Waves {
		resp.Waves = append(resp.Waves, waveToProto(&output.Waves[i]))
	}

	return resp, nil
}

func (s *Server) CreateWave(ctx context.Context, req *wms.CreateWaveRequest) (*wms.WaveResponse, error) {
	wave, err := s.waveService.CreateWave(ctx, int(req.GetMaxOrders()))
	if err != nil {
		return nil, mapErrors(err)
	}

	return waveToProto(wave), nil
}

func (s *Server) CloseWave(ctx context.Context, req *wms.CloseWaveRequest) (*wms.WaveResponse, error) {
	waveID, err := uuid.Parse(req.GetWaveId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid wave_id")
	}

	if err := s.waveService.CloseWave(ctx, waveID); err != nil {
		return nil, mapErrors(err)
	}

	wave, err := s.waveService.GetWave(ctx, waveID)
	if err != nil {
		return nil, mapErrors(err)
	}

	return waveToProto(wave), nil
}

func (s *Server) CompleteWave(ctx context.Context, req *wms.CompleteWaveRequest) (*wms.WaveResponse, error) {
	waveID, err := uuid.Parse(req.GetWaveId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid wave_id")
	}

	if err := s.waveService.CompleteWave(ctx, waveID); err != nil {
		return nil, mapErrors(err)
	}

	wave, err := s.waveService.GetWave(ctx, waveID)
	if err != nil {
		return nil, mapErrors(err)
	}

	return waveToProto(wave), nil
}

func orderToProto(order *domain.Order) *wms.OrderResponse {
	return &wms.OrderResponse{
		OrderId:   order.ID().String(),
		ChatId:    order.UserID(),
		Status:    string(order.Status()),
		Items:     itemsToProto(order.Items()),
		CreatedAt: timestamppb.New(order.CreatedAt()),
		UpdatedAt: timestamppb.New(order.UpdatedAt()),
	}
}

func waveToProto(wave *domain.Wave) *wms.WaveResponse {
	resp := &wms.WaveResponse{
		WaveId:    wave.ID().String(),
		Status:    string(wave.Status()),
		CreatedAt: timestamppb.New(wave.CreatedAt()),
	}
	for _, orderID := range wave.Orders() {
		resp.OrderIds = append(resp.OrderIds, orderID.String())
	}

	if closedAt := wave.ClosedAt(); closedAt != nil {
		resp.ClosedAt = timestamppb.New(*closedAt)
	}

	return resp
}

func itemsToProto(items []domain.OrderItem) []*wms.OrderItem {
	result := make([]*wms.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, &wms.OrderItem{
			Sku:      item.SKU,
			Quantity: int64(item.Quantity),
		})
	}

	return result
}

func itemsFromProto(items []*wms.OrderItem) []domain.OrderItem {
	result := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, domain.OrderItem{
			SKU:      item.GetSku(),
			Quantity: int(item.GetQuantity()),
		})
	}

	return result
}

func mapErrors(err error) error {
	switch {
	case errors.Is(err, application.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, application.ErrOrderNotFound), errors.Is(err, application.ErrWaveNotFound), errors.Is(err, application.ErrChatNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, application.ErrOrderCannotBeUpdated), errors.Is(err, application.ErrOrderCannotBeCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, application.ErrInvalidPageToken):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
