package domain

import "time"

type User struct {
	id        int64
	createdAt time.Time
	updatedAt time.Time
}

func NewUser(chatID int64) *User {
	now := time.Now()

	return &User{
		id:        chatID,
		createdAt: now,
		updatedAt: now,
	}
}

func UserFromExist(id int64, createdAt, updatedAt time.Time) *User {
	return &User{
		id:        id,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (u *User) ID() int64 {
	return u.id
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}
