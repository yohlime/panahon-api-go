package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateRole(t *testing.T) {
	createRandomRole(t)
}

func TestGetRole(t *testing.T) {
	role := createRandomRole(t)

	gotRole, err := testStore.GetRole(context.Background(), role.ID)
	require.NoError(t, err)
	require.NotEmpty(t, gotRole)
	requireRoleEqual(t, role, gotRole)
}

func TestListRole(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomRole(t)
	}

	arg := ListRolesParams{
		Limit:  5,
		Offset: 5,
	}

	roles, err := testStore.ListRoles(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, roles, 5)

	for _, role := range roles {
		require.NotEmpty(t, role)
	}
}

func TestUpdateRole(t *testing.T) {
	var (
		oldRole        Role
		newName        string
		newDescription string
	)

	testCases := []struct {
		name        string
		buildArg    func() UpdateRoleParams
		checkResult func(updatedRole Role, err error)
	}{
		{
			name: "OnlyName",
			buildArg: func() UpdateRoleParams {
				oldRole = createRandomRole(t)
				newName = util.RandomString(12)
				return UpdateRoleParams{
					ID: oldRole.ID,
					Name: util.NullString{Text: pgtype.Text{
						String: newName,
						Valid:  true,
					}},
				}
			},
			checkResult: func(updatedRole Role, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldRole.Name, updatedRole.Name)
				require.Equal(t, newName, updatedRole.Name)
				require.Equal(t, oldRole.Description, updatedRole.Description)
			},
		},
		{
			name: "OnlyDescription",
			buildArg: func() UpdateRoleParams {
				oldRole = createRandomRole(t)
				newDescription = util.RandomString(16)
				return UpdateRoleParams{
					ID: oldRole.ID,
					Description: util.NullString{Text: pgtype.Text{
						String: newDescription,
						Valid:  true,
					}},
				}
			},
			checkResult: func(updatedRole Role, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldRole.Description, updatedRole.Description)
				require.Equal(t, newDescription, updatedRole.Description.String())
				require.Equal(t, oldRole.Name, updatedRole.Name)
			},
		},
		{
			name: "AllFields",
			buildArg: func() UpdateRoleParams {
				var err error
				oldRole = createRandomRole(t)
				newName = util.RandomString(12)
				newDescription = util.RandomString(16)
				require.NoError(t, err)
				return UpdateRoleParams{
					ID: oldRole.ID,
					Name: util.NullString{Text: pgtype.Text{
						String: newName,
						Valid:  true,
					}},
					Description: util.NullString{Text: pgtype.Text{
						String: newDescription,
						Valid:  true,
					}},
				}
			},
			checkResult: func(updatedRole Role, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldRole.Name, updatedRole.Name)
				require.Equal(t, newName, updatedRole.Name)
				require.NotEqual(t, oldRole.Description, updatedRole.Description)
				require.Equal(t, newDescription, updatedRole.Description.String())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			updatedRole, err := testStore.UpdateRole(context.Background(), tc.buildArg())
			tc.checkResult(updatedRole, err)
		})
	}
}

func TestDeleteRole(t *testing.T) {
	role := createRandomRole(t)

	err := testStore.DeleteRole(context.Background(), role.ID)
	require.NoError(t, err)

	gotRole, err := testStore.GetRole(context.Background(), role.ID)
	require.Error(t, err)
	require.Empty(t, gotRole)
}

func createRandomRole(t *testing.T) Role {
	arg := CreateRoleParams{
		Name: util.RandomString(12),
		Description: util.NullString{
			Text: pgtype.Text{
				String: util.RandomString(16),
				Valid:  true,
			},
		},
	}

	role, err := testStore.CreateRole(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, role)

	require.Equal(t, arg.Name, role.Name)
	require.Equal(t, arg.Description, role.Description)

	require.True(t, role.UpdatedAt.Time.IsZero())
	require.True(t, role.CreatedAt.Valid)
	require.NotZero(t, role.CreatedAt.Time)

	return role
}

func requireRoleEqual(t *testing.T, r1, r2 Role) {
	require.Equal(t, r1.Name, r2.Name)
	require.Equal(t, r1.Description, r2.Description)
	require.WithinDuration(t, r1.UpdatedAt.Time, r2.UpdatedAt.Time, time.Second)
	require.WithinDuration(t, r1.CreatedAt.Time, r2.CreatedAt.Time, time.Second)
}
