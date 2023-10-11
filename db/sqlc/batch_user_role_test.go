package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BulkUserRoleTestSuite struct {
	suite.Suite
}

func TestBulkUserRoleTestSuite(t *testing.T) {
	suite.Run(t, new(BulkUserRoleTestSuite))
}

func (ts *BulkUserRoleTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *BulkUserRoleTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *BulkUserRoleTestSuite) TestBulkCreateUserRoles() {
	t := ts.T()
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
	require.Len(t, userRolesRes, nValidUserRoles)
	require.Equal(t, fmt.Sprint(userRolesRes), fmt.Sprint(userRolesParams[0:nValidUserRoles]))
	require.Len(t, errs, 1)
}

func (ts *BulkUserRoleTestSuite) TestBulkDeleteUserRoles() {
	t := ts.T()
	nRole := 5
	user := createRandomUser(t)
	roles := make([]Role, nRole)
	userRolesParams := make([]UserRolesParams, nRole)

	for i := 0; i < nRole; i++ {
		roles[i] = createRandomRole(t)

		roleName := roles[i].Name

		userRolesParams[i] = UserRolesParams{
			Username: user.Username,
			RoleName: roleName,
		}
	}

	ctx := context.Background()
	userRolesRes, errs := testStore.BulkCreateUserRoles(ctx, userRolesParams)
	require.Len(t, userRolesRes, nRole)
	require.Equal(t, fmt.Sprint(userRolesRes), fmt.Sprint(userRolesParams))
	require.Len(t, errs, 0)

	delUserRolesArg := make([]UserRolesParams, 2)
	for i, n := range []int{0, 2} {
		delUserRolesArg[i] = UserRolesParams{
			Username: user.Username,
			RoleName: userRolesRes[n].RoleName,
		}
	}
	errs = testStore.BulkDeleteUserRoles(ctx, delUserRolesArg)
	require.Len(t, errs, 0)

	newUserRoles := make([]BatchCreateUserRolesParams, 3)
	for i, n := range []int{1, 3, 4} {
		newUserRoles[i] = BatchCreateUserRolesParams{
			Username: user.Username,
			RoleName: userRolesRes[n].RoleName,
		}

	}

	retainedUserRoleNames, err := testStore.ListUserRoles(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, retainedUserRoleNames, nRole-2)

	retainedUserRoles := make([]BatchCreateUserRolesParams, len(retainedUserRoleNames))
	for r, roleName := range retainedUserRoleNames {
		retainedUserRoles[r] = BatchCreateUserRolesParams{
			Username: user.Username,
			RoleName: roleName,
		}
	}

	require.Equal(t, fmt.Sprint(retainedUserRoles), fmt.Sprint(newUserRoles))
}
