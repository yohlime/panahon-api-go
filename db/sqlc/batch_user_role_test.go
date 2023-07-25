package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBulkCreateUserRoles(t *testing.T) {
	nRole := 5
	roles := make([]Role, nRole)
	for i := 0; i < nRole; i++ {
		roles[i] = createRandomRole(t)
	}

	var userRolesParams []UserRolesParams
	nUser := 2
	users := make([]User, nUser)
	for i := 0; i < nUser; i++ {
		users[i] = createRandomUser(t)
		userRoleStart := i * 2

		for _, role := range roles[userRoleStart:] {
			roleName := role.Name

			userRolesParams = append(userRolesParams, UserRolesParams{
				Username: users[i].Username,
				RoleName: roleName,
			})
		}
	}

	userRolesParams = append(userRolesParams, UserRolesParams{
		Username: users[0].Username,
		RoleName: "INVALID",
	})

	userRolesRes, errs := testStore.BulkCreateUserRoles(context.Background(), userRolesParams)
	nValidUserRoles := len(userRolesParams) - 1
	assert.Len(t, userRolesRes, nValidUserRoles)
	assert.Equal(t, fmt.Sprint(userRolesRes), fmt.Sprint(userRolesParams[0:nValidUserRoles]))
	assert.Len(t, errs, 1)
}

func TestBulkDeleteUserRoles(t *testing.T) {
	nRole := 5
	userRolesParams := make([]UserRolesParams, nRole)
	user := createRandomUser(t)

	for i := 0; i < nRole; i++ {
		role := createRandomRole(t)

		roleName := role.Name

		userRolesParams[i] = UserRolesParams{
			Username: user.Username,
			RoleName: roleName,
		}
	}

	ctx := context.Background()
	userRolesRes, errs := testStore.BulkCreateUserRoles(ctx, userRolesParams)
	assert.Len(t, userRolesRes, nRole)
	assert.Equal(t, fmt.Sprint(userRolesRes), fmt.Sprint(userRolesParams))
	assert.Len(t, errs, 0)

	delUserRolesArg := make([]UserRolesParams, 2)
	for i, n := range []int{0, 2} {
		delUserRolesArg[i] = UserRolesParams{
			Username: user.Username,
			RoleName: userRolesRes[n].RoleName,
		}
	}
	errs = testStore.BulkDeleteUserRoles(ctx, delUserRolesArg)
	assert.Len(t, errs, 0)

	newUserRoles := make([]BatchCreateUserRolesParams, 3)
	for i, n := range []int{1, 3, 4} {
		newUserRoles[i] = BatchCreateUserRolesParams{
			Username: user.Username,
			RoleName: userRolesRes[n].RoleName,
		}

	}

	retainedUserRoleNames, err := testStore.ListUserRoles(ctx, user.ID)
	assert.NoError(t, err)
	assert.Len(t, retainedUserRoleNames, nRole-2)

	retainedUserRoles := make([]BatchCreateUserRolesParams, len(retainedUserRoleNames))
	for r, roleName := range retainedUserRoleNames {
		retainedUserRoles[r] = BatchCreateUserRolesParams{
			Username: user.Username,
			RoleName: roleName,
		}
	}

	assert.Equal(t, fmt.Sprint(retainedUserRoles), fmt.Sprint(newUserRoles))
}
