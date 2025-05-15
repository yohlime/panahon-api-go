package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MisolStationTestSuite struct {
	suite.Suite
}

func TestMisolStationTestSuite(t *testing.T) {
	suite.Run(t, new(MisolStationTestSuite))
}

func (ts *MisolStationTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *MisolStationTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *MisolStationTestSuite) TestCreateStation() {
	createRandomMisolStation(ts.T())
}

func (ts *MisolStationTestSuite) TestGetStation() {
	t := ts.T()
	stn := createRandomMisolStation(t)

	gotStn, err := testStore.GetMisolStation(context.Background(), stn.ID)

	require.NoError(t, err)
	require.NotEmpty(t, gotStn)

	require.Equal(t, stn.ID, gotStn.ID)
	require.Equal(t, stn.StationID, gotStn.StationID)
}

func (ts *MisolStationTestSuite) TestDeleteStation() {
	t := ts.T()
	stn := createRandomMisolStation(t)

	err := testStore.DeleteMisolStation(context.Background(), stn.ID)
	require.NoError(t, err)

	gotStn, err := testStore.GetMisolStation(context.Background(), stn.ID)
	require.Error(t, err)
	require.Empty(t, gotStn)
}

func createRandomMisolStation(t *testing.T) MisolStation {
	stn := createRandomStation(t, false)

	arg := CreateMisolStationParams{
		ID:        util.RandomInt[int64](1, 1000),
		StationID: stn.ID,
	}

	mStn, err := testStore.CreateMisolStation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, mStn)

	require.Equal(t, arg.StationID, mStn.StationID)
	require.Equal(t, arg.ID, mStn.ID)

	return mStn
}
