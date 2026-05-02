package application

import "errors"

var (
	ErrChatNotFound           = errors.New("chat not found")
	ErrOrderNotFound          = errors.New("order not found")
	ErrAlreadyExists          = errors.New("already exists")
	ErrOrderCannotBeUpdated   = errors.New("order cannot be updated in current status")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in current status")
	ErrInvalidPageToken       = errors.New("invalid page token")
	ErrNotFound               = errors.New("not found")
)
