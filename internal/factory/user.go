package factory

import (
	"context"
	"errors"
	"fmt"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/rs/zerolog"
)

const DefaultPassword = "password"

// UserSpec describes a user (and its roles) to seed.
type UserSpec struct {
	Username string
	FullName string
	Email    string
	Password string
	Roles    []string
}

// DefaultDevUsers returns the users seeded into the dev database.
func DefaultDevUsers() []UserSpec {
	return []UserSpec{
		{Username: "user1", FullName: "User One", Email: "user1@example.com", Password: DefaultPassword, Roles: []string{RoleUser}},
		{Username: "admin", FullName: "Admin", Email: "admin@example.com", Password: DefaultPassword, Roles: []string{RoleAdmin}},
		{Username: "admin2", FullName: "Admin Two", Email: "admin2@example.com", Password: DefaultPassword, Roles: []string{RoleSuperAdmin}},
	}
}

// CreateUser creates the user when missing and reports whether it was created.
func CreateUser(ctx context.Context, store db.Store, spec UserSpec) (db.User, bool, error) {
	user, err := store.GetUserByUsername(ctx, spec.Username)
	if err == nil {
		return user, false, nil
	}
	if !errors.Is(err, db.ErrRecordNotFound) {
		return db.User{}, false, err
	}

	hashedPassword, err := util.HashPassword(spec.Password)
	if err != nil {
		return db.User{}, false, err
	}

	created, err := store.CreateUser(ctx, db.CreateUserParams{
		Username: spec.Username,
		Password: hashedPassword,
		FullName: spec.FullName,
		Email:    spec.Email,
		Status:   models.UserStatusActive,
	})
	if err != nil {
		return db.User{}, false, err
	}

	return created, true, nil
}

// AssignRoles attaches any missing roles to the user and returns the ones added.
func AssignRoles(ctx context.Context, store db.Store, userID int64, username string, roles []string) ([]string, error) {
	existing, err := store.ListUserRoles(ctx, userID)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	has := make(map[string]bool, len(existing))
	for _, role := range existing {
		has[role] = true
	}

	var args []db.UserRolesParams
	var added []string
	for _, role := range roles {
		if has[role] {
			continue
		}
		args = append(args, db.UserRolesParams{Username: username, RoleName: role})
		added = append(added, role)
	}

	if len(args) == 0 {
		return nil, nil
	}

	if _, errs := store.BulkCreateUserRoles(ctx, args); len(errs) > 0 {
		return added, errors.Join(errs...)
	}

	return added, nil
}

// SeedUsers ensures the referenced roles, then creates/updates the users. Idempotent.
func SeedUsers(ctx context.Context, store db.Store, logger *zerolog.Logger, specs []UserSpec) error {
	for _, spec := range specs {
		for _, role := range spec.Roles {
			if _, err := EnsureRole(ctx, store, role, RoleDescriptions[role]); err != nil {
				return fmt.Errorf("ensure role %q: %w", role, err)
			}
		}
	}

	for _, spec := range specs {
		user, created, err := CreateUser(ctx, store, spec)
		if err != nil {
			return fmt.Errorf("create user %q: %w", spec.Username, err)
		}
		if created {
			logger.Info().Str("username", spec.Username).Msg("created user")
		} else {
			logger.Info().Str("username", spec.Username).Msg("user already exists")
		}

		added, err := AssignRoles(ctx, store, user.ID, user.Username, spec.Roles)
		if err != nil {
			return fmt.Errorf("assign roles to %q: %w", spec.Username, err)
		}
		if len(added) > 0 {
			logger.Info().Strs("roles", added).Str("username", spec.Username).Msg("assigned roles")
		}
	}

	return nil
}
