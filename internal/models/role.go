package models

import (
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type Role struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
} //@name Role

// NewRole creates new Role from db.Role
func NewRole(role db.Role) Role {
	res := Role{
		Name:      role.Name,
		UpdatedAt: role.UpdatedAt,
		CreatedAt: role.CreatedAt,
	}

	if role.Description.Valid {
		res.Description = role.Description.String
	}

	return res
}
