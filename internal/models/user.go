package models

import (
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
)

// User statuses. Self-registered users start pending until an admin reviews them.
const (
	UserStatusInvited  = "invited"
	UserStatusPending  = "pending"
	UserStatusActive   = "active"
	UserStatusRejected = "rejected"
)

type User struct {
	ID                int64      `json:"id" fake:"{number:1,10000}"`
	Username          string     `json:"username" fake:"{username}"`
	FullName          string     `json:"full_name" fake:"{name}"`
	Email             string     `json:"email" fake:"{email}"`
	Status            string     `json:"status" fake:"{randomstring:[invited,pending,active,rejected]}"`
	PasswordChangedAt time.Time  `json:"password_changed_at"`
	CreatedAt         time.Time  `json:"created_at"`
	ReviewedBy        *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote        string     `json:"review_note,omitempty"`
	Roles             []string   `json:"roles" fake:"{randomstring:[USER,ADMIN,SUPERADMIN]}" fakesize:"1,3"`
} //@name User

// NewUser creates new User from db.User
func NewUser(user db.User, roleNames []string) User {
	ret := User{
		ID:         user.ID,
		Username:   user.Username,
		FullName:   user.FullName,
		Status:     user.Status,
		ReviewNote: user.ReviewNote.String,
	}

	if user.Email.Valid {
		ret.Email = user.Email.String
	}

	if user.PasswordChangedAt.Valid {
		ret.PasswordChangedAt = user.PasswordChangedAt.Time
	}

	if user.CreatedAt.Valid {
		ret.CreatedAt = user.CreatedAt.Time
	}

	if user.ReviewedBy.Valid {
		v := user.ReviewedBy.Int64
		ret.ReviewedBy = &v
	}

	if user.ReviewedAt.Valid {
		t := user.ReviewedAt.Time
		ret.ReviewedAt = &t
	}

	if len(roleNames) > 0 {
		ret.Roles = roleNames
	}

	return ret
}
