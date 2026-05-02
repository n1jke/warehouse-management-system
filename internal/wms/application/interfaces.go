package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type Transactor interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event OrderEvent) error
}

type OrderRepository interface {
	Add(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domain.Order, error)
	GetByStatus(ctx context.Context, status domain.OrderStatus, limit, offset int) ([]*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type StockRepository interface {
	GetBySKUs(ctx context.Context, skus []string) (map[string]*domain.Stock, error)
	Update(ctx context.Context, stock *domain.Stock) error
}

type WaveRepository interface {
	Add(ctx context.Context, wave *domain.Wave) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Wave, error)
	GetByStatus(ctx context.Context, status domain.WaveStatus, limit, offset int) ([]*domain.Wave, error)
	Update(ctx context.Context, wave *domain.Wave) error
}

type UserRepository interface {
	Add(ctx context.Context, user *domain.User) error
	GetByChatID(ctx context.Context, chatID int64) (*domain.User, error)
}
