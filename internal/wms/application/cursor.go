package application

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Cursor struct {
	LastCreatedAt time.Time
	LastID        uuid.UUID
}

func (c Cursor) IsEmpty() bool {
	return c.LastID == uuid.Nil
}

type cursorPayload struct {
	LastCreatedAt time.Time `json:"t"`
	LastID        string    `json:"id"`
}

func EncodeCursor(c Cursor) (string, error) {
	b, err := json.Marshal(cursorPayload{
		LastCreatedAt: c.LastCreatedAt,
		LastID:        c.LastID.String(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func DecodeCursor(token string) (Cursor, error) {
	if token == "" {
		return Cursor{}, nil
	}

	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, ErrInvalidPageToken
	}

	var p cursorPayload
	if err = json.Unmarshal(b, &p); err != nil {
		return Cursor{}, ErrInvalidPageToken
	}

	id, err := uuid.Parse(p.LastID)
	if err != nil {
		return Cursor{}, ErrInvalidPageToken
	}

	return Cursor{
		LastCreatedAt: p.LastCreatedAt,
		LastID:        id,
	}, nil
}
