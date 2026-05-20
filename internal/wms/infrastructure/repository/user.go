package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

const (
	uniqueViolation string = "23505"

	addUser = `
		INSERT INTO users (id, created_at, updated_at)
		VALUES ($1, $2, $3)
	`

	getUserByChatID = `
		SELECT id, created_at, updated_at
		FROM users
		WHERE id = $1
	`
)

type UserRepo struct {
	logger *slog.Logger
	db     *pgxpool.Pool
}

func NewUserRepo(logger *slog.Logger, db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		logger: logger.With(slog.String("module", "user-repo")),
		db:     db,
	}
}

func (r *UserRepo) Add(ctx context.Context, user *domain.User) error {
	querier := GetQuerier(ctx, r.db)

	_, err := querier.Exec(ctx, addUser, user.ID(), user.CreatedAt(), user.UpdatedAt())
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			if pgErr.Code == uniqueViolation {
				return application.ErrAlreadyExists
			}
		}

		r.logger.Error("add user", slog.Any("err", err), slog.Int64("chat_id", user.ID()))
		return fmt.Errorf("add user: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByChatID(ctx context.Context, chatID int64) (*domain.User, error) {
	querier := GetQuerier(ctx, r.db)

	var (
		id                   int64
		createdAt, updatedAt time.Time
	)

	row := querier.QueryRow(ctx, getUserByChatID, chatID)
	if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrChatNotFound
		}

		r.logger.Error("get user by chat id", slog.Any("err", err), slog.Int64("chat_id", chatID))
		return nil, fmt.Errorf("get user by chatID: %w", err)
	}

	return domain.UserFromExist(id, createdAt, updatedAt), nil
}
