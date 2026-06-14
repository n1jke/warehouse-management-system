package application_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/n1jke/warehouse-management-system/internal/wms/application"
)

func TestDecodeCursor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 1, 1, 1, 1, time.UTC)
	id, err := uuid.Parse("aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa")
	require.NoError(t, err)
	validToken, err := application.EncodeCursor(application.Cursor{LastCreatedAt: now, LastID: id})
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
		check   func(*testing.T, application.Cursor)
	}{
		{
			name:  "empty token",
			token: "",
			check: func(t *testing.T, c application.Cursor) {
				assert.True(t, c.IsEmpty())
			},
		},
		{
			name:    "invalid base64",
			token:   "notbase64",
			wantErr: true,
		},
		{
			name:    "invalid json",
			token:   "dHJ1ZQ=",
			wantErr: true,
		},
		{
			name:    "invalid uuid",
			token:   "ajdfi0wNi9999990wMMMMMMMdifjnidfj",
			wantErr: true,
		},
		{
			name:  "valid cursor",
			token: validToken,
			check: func(t *testing.T, c application.Cursor) {
				assert.Equal(t, now, c.LastCreatedAt)
				assert.Equal(t, id, c.LastID)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := application.DecodeCursor(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, c)
			}
		})
	}
}
