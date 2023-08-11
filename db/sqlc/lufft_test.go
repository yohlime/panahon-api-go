package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LufftStationMsgTestSuite struct {
	suite.Suite
}

func TestLufftStationMsgTestSuite(t *testing.T) {
	suite.Run(t, new(LufftStationMsgTestSuite))
}

func (ts *LufftStationMsgTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *LufftStationMsgTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *LufftStationMsgTestSuite) TestListLufftStationMsg() {
	t := ts.T()
	station := createRandomStation(t)
	n := 10
	stnMsgs := make([]ObservationsStationhealth, n)
	for i := 0; i < n; i++ {
		stnMsgs[i] = createRandomLufftStationMsg(t, station.ID)
	}

	arg := ListLufftStationMsgParams{
		StationID: station.ID,
		Limit:     5,
		Offset:    5,
	}

	gotStnMsgs, err := testStore.ListLufftStationMsg(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotStnMsgs, 5)

	for _, msg := range gotStnMsgs {
		require.NotEmpty(t, msg)
	}
}

func (ts *LufftStationMsgTestSuite) TestCountLufftStationMsg() {
	t := ts.T()
	station := createRandomStation(t)
	n := 10
	stnMsgs := make([]ObservationsStationhealth, n)
	for i := 0; i < n; i++ {
		stnMsgs[i] = createRandomLufftStationMsg(t, station.ID)
	}

	numHealth, err := testStore.CountLufftStationMsg(context.Background(), station.ID)
	require.NoError(t, err)
	require.Equal(t, numHealth, int64(n))
}

func createRandomLufftStationMsg(t *testing.T, stationID int64) ObservationsStationhealth {
	arg := CreateStationHealthParams{
		Message: util.NullString{
			Text: pgtype.Text{
				String: util.RandomString(120),
				Valid:  true,
			},
		},
		Timestamp: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		StationID: stationID,
	}

	health, err := testStore.CreateStationHealth(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, health)

	require.Equal(t, arg.Message, health.Message)
	require.Equal(t, arg.StationID, health.StationID)

	require.True(t, health.UpdatedAt.Time.IsZero())
	require.True(t, health.CreatedAt.Valid)
	require.NotZero(t, health.CreatedAt.Time)

	return health
}
