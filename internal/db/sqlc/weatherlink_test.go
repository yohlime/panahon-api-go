package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WeatherlinkTestSuite struct {
	suite.Suite
}

func TestWeatherlinkTestSuite(t *testing.T) {
	suite.Run(t, new(WeatherlinkTestSuite))
}

func (ts *WeatherlinkTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *WeatherlinkTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *WeatherlinkTestSuite) TestCreateWeatherlinkStation() {
	createRandomWeatherlinkStation(ts.T(), "Default")
	createRandomWeatherlinkStation(ts.T(), "V2")
}

func (ts *WeatherlinkTestSuite) TestListWeatherlinkStations() {
	t := ts.T()
	n := 10
	for i := 0; i < n; i++ {
		createRandomWeatherlinkStation(t, "Default")
	}

	arg := ListWeatherlinkStationsParams{
		Limit:  pgtype.Int4{Int32: 5, Valid: true},
		Offset: 5,
	}

	gotStns, err := testStore.ListWeatherlinkStations(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotStns, 5)

	for _, stn := range gotStns {
		require.NotEmpty(t, stn)
	}
}

func createRandomWeatherlinkStation(t *testing.T, stationType string) Weatherlink {
	stn := createRandomStation(t, false)
	arg := CreateWeatherlinkStationParams{
		StationID: stn.ID,
	}

	if stationType == "V2" {
		arg.ApiKey = pgtype.Text{
			String: util.RandomString(12),
			Valid:  true,
		}
		arg.ApiSecret = pgtype.Text{
			String: util.RandomString(24),
			Valid:  true,
		}
	} else {
		arg.Uuid = pgtype.Text{
			String: util.RandomString(24),
			Valid:  true,
		}
	}

	wl, err := testStore.CreateWeatherlinkStation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, wl)

	if stationType == "V2" {
		require.True(t, wl.ApiKey.Valid)
		require.Equal(t, arg.ApiKey.String, wl.ApiKey.String)
		require.True(t, wl.ApiSecret.Valid)
		require.Equal(t, arg.ApiSecret.String, wl.ApiSecret.String)
	} else {
		require.True(t, wl.Uuid.Valid)
		require.Equal(t, arg.Uuid.String, wl.Uuid.String)
	}

	return wl
}
