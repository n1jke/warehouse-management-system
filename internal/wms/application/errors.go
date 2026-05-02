package application

import "errors"

var (
	ErrChatNotFound           = errors.New("chat not found")
	ErrAlreadyExists          = errors.New("already exists")
	ErrOrderCannotBeUpdated   = errors.New("order cannot be updated in current status")
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in current status")
	ErrInvalidPageToken       = errors.New("invalid page token")
)
