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

type ObservationTestSuite struct {
	suite.Suite
}

func TestObservationTestSuite(t *testing.T) {
	suite.Run(t, new(ObservationTestSuite))
}

func (ts *ObservationTestSuite) SetupTest() {
	util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
}

func (ts *ObservationTestSuite) TearDownTest() {
	runDBMigrationDown(testConfig.MigrationPath, testConfig.DBSource)
}

func (ts *ObservationTestSuite) TestCreateStationObservation() {
	t := ts.T()
	station := createRandomStation(t)
	createRandomObservation(t, station.ID)
}

func (ts *ObservationTestSuite) TestGetStationObservation() {
	t := ts.T()
	station := createRandomStation(t)
	obs := createRandomObservation(t, station.ID)

	arg := GetStationObservationParams{
		StationID: obs.StationID,
		ID:        obs.ID,
	}
	gotObs, err := testStore.GetStationObservation(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, gotObs)

	require.Equal(t, obs, gotObs)
}

func (ts *ObservationTestSuite) TestListStationObservations() {
	t := ts.T()
	station := createRandomStation(t)
	n := 10
	for i := 0; i < n; i++ {
		createRandomObservation(t, station.ID)
	}

	arg := ListStationObservationsParams{
		StationID: station.ID,
		Limit:     5,
		Offset:    5,
	}

	gotObservations, err := testStore.ListStationObservations(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotObservations, 5)

	for _, obs := range gotObservations {
		require.NotEmpty(t, obs)
	}
}

func (ts *ObservationTestSuite) TestCountStationObservations() {
	t := ts.T()
	station := createRandomStation(t)
	n := 10
	for i := 0; i < n; i++ {
		createRandomObservation(t, station.ID)
	}

	numObservations, err := testStore.CountStationObservations(context.Background(), station.ID)
	require.NoError(t, err)
	require.Equal(t, numObservations, int64(n))
}

func (ts *ObservationTestSuite) TestUpdateStationObservation() {
	var (
		oldObs  ObservationsObservation
		newPres util.NullFloat4
		newRr   util.NullFloat4
	)

	t := ts.T()

	station := createRandomStation(t)

	testCases := []struct {
		name        string
		buildArg    func() UpdateStationObservationParams
		checkResult func(updatedObs ObservationsObservation, err error)
	}{
		{
			name: "SomeValues",
			buildArg: func() UpdateStationObservationParams {
				oldObs = createRandomObservation(t, station.ID)
				newPres = util.RandomNullFloat4(995.0, 1100.0)
				newRr = util.RandomNullFloat4(0.0, 15.0)

				return UpdateStationObservationParams{
					StationID: station.ID,
					ID:        oldObs.ID,
					Pres:      newPres,
					Rr:        newRr,
				}
			},
			checkResult: func(updatedObs ObservationsObservation, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldObs.Pres, updatedObs.Pres)
				require.NotEqual(t, oldObs.Rr, updatedObs.Rr)
				require.Equal(t, oldObs.Temp, updatedObs.Temp)

				require.True(t, updatedObs.UpdatedAt.Valid)
				require.NotZero(t, updatedObs.UpdatedAt.Time)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			updatedObs, err := testStore.UpdateStationObservation(context.Background(), tc.buildArg())
			tc.checkResult(updatedObs, err)
		})
	}
}

func (ts *ObservationTestSuite) TestDeleteStationObservation() {
	t := ts.T()
	station := createRandomStation(t)
	obs := createRandomObservation(t, station.ID)

	arg := DeleteStationObservationParams{
		StationID: obs.StationID,
		ID:        obs.ID,
	}

	err := testStore.DeleteStationObservation(context.Background(), arg)
	require.NoError(t, err)

	getArg := GetStationObservationParams(arg)
	gotObs, err := testStore.GetStationObservation(context.Background(), getArg)
	require.Error(t, err)
	require.Empty(t, gotObs)
}

func createRandomObservation(t *testing.T, stationID int64) ObservationsObservation {
	arg := CreateStationObservationParams{
		Pres: util.RandomNullFloat4(990.0, 1100.0),
		Temp: util.RandomNullFloat4(16.0, 38.0),
		Rr:   util.RandomNullFloat4(0.0, 10.0),
		Timestamp: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		StationID: stationID,
	}

	obs, err := testStore.CreateStationObservation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, obs)

	require.Equal(t, arg.Pres, obs.Pres)
	require.Equal(t, arg.Temp, obs.Temp)
	require.Equal(t, arg.Rr, obs.Rr)
	require.Equal(t, arg.StationID, obs.StationID)

	require.True(t, obs.UpdatedAt.Time.IsZero())
	require.True(t, obs.CreatedAt.Valid)
	require.NotZero(t, obs.CreatedAt.Time)

	return obs
}
