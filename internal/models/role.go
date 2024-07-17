package models

import (
	"fmt"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
)

type Role struct {
	Name        string    `json:"name" fake:"{regex:[A-Z]{6,12}}"`
	Description string    `json:"description" fake:"{sentence:10}"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
} //@name Role

// NewRole creates new Role from db.Role
func NewRole(role db.Role) Role {
	res := Role{
		Name: role.Name,
	}

	if role.Description.Valid {
		res.Description = role.Description.String
	}

	if role.CreatedAt.Valid {
		res.CreatedAt = role.CreatedAt.Time
	}
	if role.UpdatedAt.Valid {
		res.UpdatedAt = role.UpdatedAt.Time
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
