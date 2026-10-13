package factory

import (
	"context"
	"testing"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEnsureRole(t *testing.T) {
	ctx := context.Background()

	t.Run("existing", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		want := db.Role{ID: 1, Name: RoleUser}
		store.EXPECT().GetRoleByName(mock.Anything, RoleUser).Return(want, nil)

		got, err := EnsureRole(ctx, store, RoleUser, "User role")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("create", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().GetRoleByName(mock.Anything, RoleAdmin).Return(db.Role{}, db.ErrRecordNotFound)
		want := db.Role{ID: 2, Name: RoleAdmin}
		store.EXPECT().CreateRole(mock.Anything, mock.MatchedBy(func(arg db.CreateRoleParams) bool {
			return arg.Name == RoleAdmin
		})).Return(want, nil)

		got, err := EnsureRole(ctx, store, RoleAdmin, "Admin role")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	spec := UserSpec{Username: "user1", FullName: "User One", Email: "user1@example.com", Password: "password"}

	t.Run("existing", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		want := db.User{ID: 1, Username: "user1"}
		store.EXPECT().GetUserByUsername(mock.Anything, "user1").Return(want, nil)

		got, created, err := CreateUser(ctx, store, spec)
		require.NoError(t, err)
		require.False(t, created)
		require.Equal(t, want, got)
	})

	t.Run("create hashes password", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().GetUserByUsername(mock.Anything, "user1").Return(db.User{}, db.ErrRecordNotFound)
		store.EXPECT().CreateUser(mock.Anything, mock.MatchedBy(func(arg db.CreateUserParams) bool {
			return arg.Username == "user1" && arg.Password != "" && arg.Password != "password"
		})).Return(db.User{ID: 1, Username: "user1"}, nil)

		got, created, err := CreateUser(ctx, store, spec)
		require.NoError(t, err)
		require.True(t, created)
		require.Equal(t, int64(1), got.ID)
	})
}

func TestAssignRoles(t *testing.T) {
	ctx := context.Background()

	t.Run("adds only missing", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().ListUserRoles(mock.Anything, int64(1)).Return([]string{RoleUser}, nil)
		store.EXPECT().BulkCreateUserRoles(mock.Anything, mock.MatchedBy(func(args []db.UserRolesParams) bool {
			return len(args) == 1 && args[0].Username == "user1" && args[0].RoleName == RoleAdmin
		})).Return([]db.UserRolesParams{{Username: "user1", RoleName: RoleAdmin}}, nil)

		added, err := AssignRoles(ctx, store, 1, "user1", []string{RoleUser, RoleAdmin})
		require.NoError(t, err)
		require.Equal(t, []string{RoleAdmin}, added)
	})

	t.Run("no roles", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().ListUserRoles(mock.Anything, int64(1)).Return(nil, db.ErrRecordNotFound)

		added, err := AssignRoles(ctx, store, 1, "user1", nil)
		require.NoError(t, err)
		require.Nil(t, added)
	})
}

func TestSeedUsers(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()
	specs := []UserSpec{{Username: "user1", FullName: "User One", Email: "user1@example.com", Password: "password", Roles: []string{RoleUser}}}

	store := mockdb.NewMockStore(t)
	store.EXPECT().GetRoleByName(mock.Anything, RoleUser).Return(db.Role{ID: 1, Name: RoleUser}, nil)
	store.EXPECT().GetUserByUsername(mock.Anything, "user1").Return(db.User{}, db.ErrRecordNotFound)
	store.EXPECT().CreateUser(mock.Anything, mock.Anything).Return(db.User{ID: 1, Username: "user1"}, nil)
	store.EXPECT().ListUserRoles(mock.Anything, int64(1)).Return(nil, db.ErrRecordNotFound)
	store.EXPECT().BulkCreateUserRoles(mock.Anything, mock.Anything).Return([]db.UserRolesParams{{Username: "user1", RoleName: RoleUser}}, nil)

	require.NoError(t, SeedUsers(ctx, store, &logger, specs))
}
