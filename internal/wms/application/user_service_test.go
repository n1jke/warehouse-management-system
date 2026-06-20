package application_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
	"github.com/n1jke/warehouse-management-system/internal/wms/application/mocks"
	"github.com/n1jke/warehouse-management-system/internal/wms/domain"
)

func TestRegisterUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		chatID  int64
		prepare func(*mocks.MockUserRepository)
		wantErr error
	}{
		{
			name:   "success",
			chatID: 999,
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:   "already exists",
			chatID: 999,
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(application.ErrAlreadyExists)
			},
			wantErr: application.ErrAlreadyExists,
		},
		{
			name:   "repo error",
			chatID: 999,
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().Add(gomock.Any(), gomock.Any()).Return(assert.AnError)
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
			userRepo := mocks.NewMockUserRepository(ctrl)
			tt.prepare(userRepo)

			svc := application.NewUserService(logger, userRepo)

			user, err := svc.RegisterUser(context.Background(), tt.chatID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.chatID, user.ID())
		})
	}
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	chatID := int64(999)

	tests := []struct {
		name    string
		prepare func(*mocks.MockUserRepository)
		wantErr error
	}{
		{
			name: "success",
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().GetByChatID(gomock.Any(), chatID).Return(domain.NewUser(chatID), nil)
			},
		},
		{
			name: "chat not found",
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().GetByChatID(gomock.Any(), chatID).Return(nil, application.ErrChatNotFound)
			},
			wantErr: application.ErrChatNotFound,
		},
		{
			name: "repo error",
			prepare: func(r *mocks.MockUserRepository) {
				r.EXPECT().GetByChatID(gomock.Any(), chatID).Return(nil, assert.AnError)
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
			userRepo := mocks.NewMockUserRepository(ctrl)
			tt.prepare(userRepo)

			svc := application.NewUserService(logger, userRepo)

			user, err := svc.GetUser(context.Background(), chatID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, chatID, user.ID())
		})
	}
}
