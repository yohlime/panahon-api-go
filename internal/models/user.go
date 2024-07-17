package models

import (
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
)

type User struct {
	Username          string    `json:"username" fake:"{username}"`
	FullName          string    `json:"full_name" fake:"{name}"`
	Email             string    `json:"email" fake:"{email}"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
	Roles             []string  `json:"roles" fake:"{randomstring:[USER,ADMIN,SUPERADMIN]}" fakesize:"1,3"`
} //@name User

// NewUser creates new User from db.User
func NewUser(user db.User, roleNames []string) User {
	ret := User{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	}

	if user.PasswordChangedAt.Valid {
		ret.PasswordChangedAt = user.PasswordChangedAt.Time
	}

	if user.CreatedAt.Valid {
		ret.CreatedAt = user.CreatedAt.Time
	}

	if len(roleNames) > 0 {
		ret.Roles = roleNames
	}

	return ret
}
