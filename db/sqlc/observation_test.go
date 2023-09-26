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
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *ObservationTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *ObservationTestSuite) TestCreateStationObservation() {
	t := ts.T()
	station := createRandomStation(t, false)
	createRandomObservation(t, station.ID)
}

func (ts *ObservationTestSuite) TestGetStationObservation() {
	t := ts.T()
	station := createRandomStation(t, false)
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
	timeNow := time.Now()
	n := 10
	station := createRandomStation(t, false)
	for j := 0; j < n; j++ {
		createRandomObservation(t, station.ID)
	}

	testCases := []struct {
		name   string
		arg    ListStationObservationsParams
		result func(obs []ObservationsObservation, err error)
	}{
		{
			name: "Default",
			arg: ListStationObservationsParams{
				StationID: station.ID,
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, n)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimit",
			arg: ListStationObservationsParams{
				StationID: station.ID,
				Limit: util.NullInt4{
					Int4: pgtype.Int4{
						Int32: 5,
						Valid: true,
					},
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 5)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimitAndOffset",
			arg: ListStationObservationsParams{
				StationID: station.ID,
				Limit: util.NullInt4{
					Int4: pgtype.Int4{
						Int32: 5,
						Valid: true,
					},
				},
				Offset: 7,
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 3)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: ListStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, n)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithPastDate",
			arg: ListStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -4),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
		{
			name: "WithFutureDate",
			arg: ListStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 4),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		gotObservations, err := testStore.ListStationObservations(context.Background(), tc.arg)
		tc.result(gotObservations, err)
	}
}

func (ts *ObservationTestSuite) TestCountStationObservations() {
	t := ts.T()
	timeNow := time.Now()
	n := 10
	station := createRandomStation(t, false)
	for j := 0; j < n; j++ {
		createRandomObservation(t, station.ID)
	}

	testCases := []struct {
		name   string
		arg    CountStationObservationsParams
		result func(count int64, err error)
	}{
		{
			name: "Default",
			arg: CountStationObservationsParams{
				StationID: station.ID,
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(n), count)
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: CountStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(n), count)
			},
		},
		{
			name: "WithPastDate",
			arg: CountStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -4),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), count)
			},
		},
		{
			name: "WithFutureDate",
			arg: CountStationObservationsParams{
				StationID:   station.ID,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 4),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), count)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		numObservations, err := testStore.CountStationObservations(context.Background(), tc.arg)
		tc.result(numObservations, err)
	}
}

func (ts *ObservationTestSuite) TestListObservations() {
	t := ts.T()
	timeNow := time.Now()
	n, m := 5, 10
	stations := make([]ObservationsStation, n)
	var selStationIDs []int64
	for i := range stations {
		stations[i] = createRandomStation(t, false)
		for j := 0; j < m; j++ {
			createRandomObservation(t, stations[i].ID)
		}
		if i%2 == 0 {
			selStationIDs = append(selStationIDs, stations[i].ID)
		}
	}

	testCases := []struct {
		name   string
		arg    ListObservationsParams
		result func(obs []ObservationsObservation, err error)
	}{
		{
			name: "Default",
			arg: ListObservationsParams{
				StationIds: selStationIDs,
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, m*len(selStationIDs))

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimit",
			arg: ListObservationsParams{
				StationIds: selStationIDs,
				Limit: util.NullInt4{
					Int4: pgtype.Int4{
						Int32: 5,
						Valid: true,
					},
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 5)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimitAndOffset",
			arg: ListObservationsParams{
				StationIds: selStationIDs,
				Limit: util.NullInt4{
					Int4: pgtype.Int4{
						Int32: 5,
						Valid: true,
					},
				},
				Offset: int32(m*len(selStationIDs) - 3),
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 3)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: ListObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, m*len(selStationIDs))

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithPastDate",
			arg: ListObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -4),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
		{
			name: "WithFutureDate",
			arg: ListObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 4),
					Valid: true,
				},
			},
			result: func(obs []ObservationsObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		gotObservations, err := testStore.ListObservations(context.Background(), tc.arg)
		tc.result(gotObservations, err)
	}
}

func (ts *ObservationTestSuite) TestCountObservations() {
	t := ts.T()
	timeNow := time.Now()
	n, m := 5, 10
	stations := make([]ObservationsStation, n)
	var selStationIDs []int64
	for i := range stations {
		stations[i] = createRandomStation(t, false)
		for j := 0; j < m; j++ {
			createRandomObservation(t, stations[i].ID)
		}
		if i%2 == 0 {
			selStationIDs = append(selStationIDs, stations[i].ID)
		}
	}

	testCases := []struct {
		name   string
		arg    CountObservationsParams
		result func(count int64, err error)
	}{
		{
			name: "Default",
			arg: CountObservationsParams{
				StationIds: selStationIDs,
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(m*len(selStationIDs)), count)
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: CountObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(m*len(selStationIDs)), count)
			},
		},
		{
			name: "WithPastDate",
			arg: CountObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -4),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, -2),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), count)
			},
		},
		{
			name: "WithFutureDate",
			arg: CountObservationsParams{
				StationIds:  selStationIDs,
				IsStartDate: true,
				StartDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 2),
					Valid: true,
				},
				IsEndDate: true,
				EndDate: pgtype.Timestamptz{
					Time:  timeNow.AddDate(0, 0, 4),
					Valid: true,
				},
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(0), count)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		numObservations, err := testStore.CountObservations(context.Background(), tc.arg)
		tc.result(numObservations, err)
	}
}

func (ts *ObservationTestSuite) TestUpdateStationObservation() {
	var (
		oldObs  ObservationsObservation
		newPres util.NullFloat4
		newRr   util.NullFloat4
	)

	t := ts.T()

	station := createRandomStation(t, false)

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
	station := createRandomStation(t, false)
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
