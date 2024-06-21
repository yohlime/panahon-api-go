package models

import (
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	Username          string             `json:"username"`
	FullName          string             `json:"full_name"`
	Email             string             `json:"email"`
	PasswordChangedAt pgtype.Timestamptz `json:"password_changed_at"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	Roles             []string           `json:"roles"`
} //@name User

// NewUser creates new User from db.User
func NewUser(user db.User, roleNames []string) User {
	ret := User{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}

	if len(roleNames) > 0 {
		ret.Roles = roleNames
	}

	return ret
}
