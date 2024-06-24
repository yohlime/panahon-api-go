package models

import (
	"fmt"

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

type AdminRoleType string

const (
	AdminRole      AdminRoleType = "ADMIN"
	SuperAdminRole AdminRoleType = "SUPERADMIN"
)

func (rt AdminRoleType) IsValid() error {
	switch rt {
	case AdminRole, SuperAdminRole:
		return nil
	}
	return fmt.Errorf("invalid admin_role type")
}

func IsAdminRole(role string) bool {
	rt := AdminRoleType(role)
	if err := rt.IsValid(); err != nil {
		return false
	}
	return true
}
