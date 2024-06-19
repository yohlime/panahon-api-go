package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	suite.Suite
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}

func (ts *UserTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *UserTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *UserTestSuite) TestCreateUser() {
	createRandomUser(ts.T())
}

func (ts *UserTestSuite) TestGetUser() {
	t := ts.T()
	user := createRandomUser(t)

	gotUser, err := testStore.GetUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, gotUser)
	requireUserEqual(t, user, gotUser)
}

func (ts *UserTestSuite) TestListUsers() {
	t := ts.T()
	n := 10
	for i := 0; i < n; i++ {
		createRandomUser(t)
	}

	arg := ListUsersParams{
		Limit:  5,
		Offset: 5,
	}

	gotUsers, err := testStore.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotUsers, 5)

	for _, user := range gotUsers {
		require.NotEmpty(t, user)
	}
}

func (ts *UserTestSuite) TestCountUsers() {
	t := ts.T()
	n := 10
	for i := 0; i < n; i++ {
		createRandomUser(t)
	}

	numUsers, err := testStore.CountUsers(context.Background())
	require.NoError(t, err)
	require.Equal(t, numUsers, int64(n))
}

func (ts *UserTestSuite) TestUpdateUser() {
	var (
		oldUser           User
		newFullName       string
		newEmail          string
		newHashedPassword string
	)

	t := ts.T()

	testCases := []struct {
		name        string
		buildArg    func() UpdateUserParams
		checkResult func(updatedUser User, err error)
	}{
		{
			name: "OnlyFullName",
			buildArg: func() UpdateUserParams {
				oldUser = createRandomUser(t)
				newFullName = util.RandomString(12)
				return UpdateUserParams{
					ID: oldUser.ID,
					FullName: pgtype.Text{
						String: newFullName,
						Valid:  true,
					},
				}
			},
			checkResult: func(updatedUser User, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldUser.FullName, updatedUser.FullName)
				require.Equal(t, newFullName, updatedUser.FullName)
				require.Equal(t, oldUser.Email, updatedUser.Email)
				require.Equal(t, oldUser.Password, updatedUser.Password)
			},
		},
		{
			name: "OnlyEmail",
			buildArg: func() UpdateUserParams {
				oldUser = createRandomUser(t)
				newEmail = util.RandomEmail()
				return UpdateUserParams{
					ID: oldUser.ID,
					Email: pgtype.Text{
						String: newEmail,
						Valid:  true,
					},
				}
			},
			checkResult: func(updatedUser User, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldUser.Email, updatedUser.Email)
				require.Equal(t, newEmail, updatedUser.Email)
				require.Equal(t, oldUser.FullName, updatedUser.FullName)
				require.Equal(t, oldUser.Password, updatedUser.Password)
			},
		},
		{
			name: "OnlyPassword",
			buildArg: func() UpdateUserParams {
				var err error
				oldUser = createRandomUser(t)
				newPassword := util.RandomString(6)
				newHashedPassword, err = util.HashPassword(newPassword)
				require.NoError(t, err)
				return UpdateUserParams{
					ID: oldUser.ID,
					Password: pgtype.Text{
						String: newHashedPassword,
						Valid:  true,
					},
				}
			},
			checkResult: func(updatedUser User, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldUser.Password, updatedUser.Password)
				require.Equal(t, newHashedPassword, updatedUser.Password)
				require.Equal(t, oldUser.FullName, updatedUser.FullName)
				require.Equal(t, oldUser.Email, updatedUser.Email)
			},
		},
		{
			name: "AllFields",
			buildArg: func() UpdateUserParams {
				var err error
				oldUser = createRandomUser(t)
				newFullName = util.RandomString(12)
				newEmail = util.RandomEmail()
				newPassword := util.RandomString(6)
				newHashedPassword, err = util.HashPassword(newPassword)
				require.NoError(t, err)
				return UpdateUserParams{
					ID: oldUser.ID,
					FullName: pgtype.Text{
						String: newFullName,
						Valid:  true,
					},
					Email: pgtype.Text{
						String: newEmail,
						Valid:  true,
					},
					Password: pgtype.Text{
						String: newHashedPassword,
						Valid:  true,
					},
				}
			},
			checkResult: func(updatedUser User, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldUser.Password, updatedUser.Password)
				require.Equal(t, newHashedPassword, updatedUser.Password)
				require.NotEqual(t, oldUser.Email, updatedUser.Email)
				require.Equal(t, newEmail, updatedUser.Email)
				require.NotEqual(t, oldUser.FullName, updatedUser.FullName)
				require.Equal(t, newFullName, updatedUser.FullName)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			updatedUser, err := testStore.UpdateUser(context.Background(), tc.buildArg())
			tc.checkResult(updatedUser, err)
		})
	}
}

func (ts *UserTestSuite) TestDeleteUser() {
	t := ts.T()
	user := createRandomUser(t)

	err := testStore.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err)

	gotUser, err := testStore.GetUser(context.Background(), user.ID)
	require.Error(t, err)
	require.Empty(t, gotUser)
}

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		Username: util.RandomString(12),
		Password: hashedPassword,
		FullName: util.RandomString(12),
		Email:    util.RandomEmail(),
	}

	user, err := testStore.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.Password, user.Password)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)
	require.True(t, user.PasswordChangedAt.Time.IsZero())
	require.True(t, user.CreatedAt.Valid)
	require.NotZero(t, user.CreatedAt.Time)

	return user
}

func requireUserEqual(t *testing.T, u1, u2 User) {
	require.Equal(t, u1.Username, u2.Username)
	require.Equal(t, u1.Password, u2.Password)
	require.Equal(t, u1.FullName, u2.FullName)
	require.Equal(t, u1.Email, u2.Email)
	require.WithinDuration(t, u1.PasswordChangedAt.Time, u2.PasswordChangedAt.Time, time.Second)
	require.WithinDuration(t, u1.CreatedAt.Time, u2.CreatedAt.Time, time.Second)
}
