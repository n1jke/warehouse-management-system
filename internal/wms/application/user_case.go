package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

type UserService struct {
	logger   *slog.Logger
	userRepo UserRepository
}

func NewUserService(logger *slog.Logger, userRepo UserRepository) *UserService {
	return &UserService{
		logger:   logger.With(slog.String("module", "user-service")),
		userRepo: userRepo,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, chatID int64) (*domain.User, error) {
	user := domain.NewUser(chatID)

	if err := s.userRepo.Add(ctx, user); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return nil, ErrAlreadyExists
		}

		s.logger.Error("add user", slog.Int64("chatID", chatID), slog.Any("err", err))

		return nil, fmt.Errorf("add user: %w", err)
	}

	s.logger.Info("user register successfully", slog.Int64("chatID", chatID))

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, chatID int64) (*domain.User, error) {
	user, err := s.userRepo.GetByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrChatNotFound
		}

		s.logger.Error("get user by chatID", slog.Int64("chatID", chatID), slog.Any("err", err))

		return nil, fmt.Errorf("get user by chatID: %w", err)
	}

	return user, nil
}
