package application

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderCursor struct {
	LastCreatedAt time.Time
	LastID        uuid.UUID
}

func (c OrderCursor) IsEmpty() bool {
	return c.LastID == uuid.Nil
}

type cursorPayload struct {
	LastCreatedAt time.Time `json:"t"`
	LastID        string    `json:"id"`
}

func encodeCursor(c OrderCursor) (string, error) {
	b, err := json.Marshal(cursorPayload{
		LastCreatedAt: c.LastCreatedAt,
		LastID:        c.LastID.String(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func decodeCursor(token string) (OrderCursor, error) {
	if token == "" {
		return OrderCursor{}, nil
	}

	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return OrderCursor{}, ErrInvalidPageToken
	}

	var p cursorPayload
	if err = json.Unmarshal(b, &p); err != nil {
		return OrderCursor{}, ErrInvalidPageToken
	}

	id, err := uuid.Parse(p.LastID)
	if err != nil {
		return OrderCursor{}, ErrInvalidPageToken
	}

	return OrderCursor{
		LastCreatedAt: p.LastCreatedAt,
		LastID:        id,
	}, nil
}
