package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SessionTestSuite struct {
	suite.Suite
}

func TestSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}

func (ts *SessionTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *SessionTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *SessionTestSuite) TestCreateSession() {
	createRandomSession(ts.T())
}

func (ts *SessionTestSuite) TestGetSession() {
	t := ts.T()
	session := createRandomSession(t)

	gotSession, err := testStore.GetSession(context.Background(), session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, gotSession)
	requireSessionEqual(t, session, gotSession)
}

func (ts *SessionTestSuite) TestDeleteSession() {
	t := ts.T()
	session := createRandomSession(t)

	err := testStore.DeleteSession(context.Background(), session.ID)
	require.NoError(t, err)

	gotUser, err := testStore.GetSession(context.Background(), session.ID)
	require.Error(t, err)
	require.Empty(t, gotUser)
}

func createRandomSession(t *testing.T) Session {
	user := createRandomUser(t)
	arg := CreateSessionParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	session, err := testStore.CreateSession(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, session)

	return session
}

func requireSessionEqual(t *testing.T, sess1, sess2 Session) {
	require.Equal(t, sess1.UserID, sess2.UserID)
	require.WithinDuration(t, sess1.ExpiresAt.Time, sess2.ExpiresAt.Time, time.Second)
}
