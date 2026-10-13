package factory

import (
	"context"
	"errors"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
)

const (
	RoleUser       = "USER"
	RoleAdmin      = "ADMIN"
	RoleSuperAdmin = "SUPERADMIN"
)

// RoleDescriptions maps known role names to their descriptions.
var RoleDescriptions = map[string]string{
	RoleUser:       "User role",
	RoleAdmin:      "Admin role",
	RoleSuperAdmin: "Superadmin role",
}

// EnsureRole fetches the role by name, creating it when it does not exist yet.
func EnsureRole(ctx context.Context, store db.Store, name, description string) (db.Role, error) {
	role, err := store.GetRoleByName(ctx, name)
	if err == nil {
		return role, nil
	}
	if !errors.Is(err, db.ErrRecordNotFound) {
		return db.Role{}, err
	}

	return store.CreateRole(ctx, db.CreateRoleParams{
		Name:        name,
		Description: util.ToPgText(description),
	})
}
