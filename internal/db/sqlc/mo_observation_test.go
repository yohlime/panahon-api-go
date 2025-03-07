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

type MOObservationTestSuite struct {
	suite.Suite
}

func TestMOObservationTestSuite(t *testing.T) {
	suite.Run(t, new(MOObservationTestSuite))
}

func (ts *MOObservationTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *MOObservationTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *MOObservationTestSuite) TestCreateStationMOObservation() {
	t := ts.T()
	station := createRandomStation(t, false)
	createRandomMOObservation(t, station.ID)
}

func (ts *MOObservationTestSuite) TestGetStationMOObservation() {
	t := ts.T()
	station := createRandomStation(t, false)
	obs := createRandomMOObservation(t, station.ID)

	arg := GetStationMOObservationParams{
		StationID: obs.StationID,
		ID:        obs.ID,
	}
	gotObs, err := testStore.GetStationMOObservation(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, gotObs)

	require.Equal(t, obs, gotObs)
}

func (ts *MOObservationTestSuite) TestListStationMOObservations() {
	t := ts.T()
	timeNow := time.Now()
	n := 10
	station := createRandomStation(t, false)
	for j := 0; j < n; j++ {
		createRandomMOObservation(t, station.ID)
	}

	testCases := []struct {
		name   string
		arg    ListStationMOObservationsParams
		result func(obs []ObservationsMoObservation, err error)
	}{
		{
			name: "Default",
			arg: ListStationMOObservationsParams{
				StationID: station.ID,
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, n)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimit",
			arg: ListStationMOObservationsParams{
				StationID: station.ID,
				Limit: pgtype.Int4{
					Int32: 5,
					Valid: true,
				},
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 5)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimitAndOffset",
			arg: ListStationMOObservationsParams{
				StationID: station.ID,
				Limit: pgtype.Int4{
					Int32: 5,
					Valid: true,
				},
				Offset: 7,
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 3)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: ListStationMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, n)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithPastDate",
			arg: ListStationMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
		{
			name: "WithFutureDate",
			arg: ListStationMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		gotObservations, err := testStore.ListStationMOObservations(context.Background(), tc.arg)
		tc.result(gotObservations, err)
	}
}

func (ts *MOObservationTestSuite) TestCountStationMOObservations() {
	t := ts.T()
	timeNow := time.Now()
	n := 10
	station := createRandomStation(t, false)
	for j := 0; j < n; j++ {
		createRandomMOObservation(t, station.ID)
	}

	testCases := []struct {
		name   string
		arg    CountStationMOObservationsParams
		result func(count int64, err error)
	}{
		{
			name: "Default",
			arg: CountStationMOObservationsParams{
				StationID: station.ID,
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(n), count)
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: CountStationMOObservationsParams{
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
			arg: CountStationMOObservationsParams{
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
			arg: CountStationMOObservationsParams{
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
		numObservations, err := testStore.CountStationMOObservations(context.Background(), tc.arg)
		tc.result(numObservations, err)
	}
}

func (ts *MOObservationTestSuite) TestListObservations() {
	t := ts.T()
	timeNow := time.Now()
	n, m := 5, 10
	stations := make([]ObservationsStation, n)
	var selStationIDs []int64
	for i := range stations {
		stations[i] = createRandomStation(t, false)
		for j := 0; j < m; j++ {
			createRandomMOObservation(t, stations[i].ID)
		}
		if i%2 == 0 {
			selStationIDs = append(selStationIDs, stations[i].ID)
		}
	}

	testCases := []struct {
		name   string
		arg    ListMOObservationsParams
		result func(obs []ObservationsMoObservation, err error)
	}{
		{
			name: "Default",
			arg: ListMOObservationsParams{
				StationIds: selStationIDs,
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, m*len(selStationIDs))

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimit",
			arg: ListMOObservationsParams{
				StationIds: selStationIDs,
				Limit: pgtype.Int4{
					Int32: 5,
					Valid: true,
				},
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 5)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithLimitAndOffset",
			arg: ListMOObservationsParams{
				StationIds: selStationIDs,
				Limit: pgtype.Int4{
					Int32: 5,
					Valid: true,
				},
				Offset: int32(m*len(selStationIDs) - 3),
			},
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, 3)

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: ListMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Len(t, obs, m*len(selStationIDs))

				for _, obs := range obs {
					require.NotEmpty(t, obs)
				}
			},
		},
		{
			name: "WithPastDate",
			arg: ListMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
		{
			name: "WithFutureDate",
			arg: ListMOObservationsParams{
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
			result: func(obs []ObservationsMoObservation, err error) {
				require.NoError(t, err)
				require.Empty(t, obs, 0)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		gotObservations, err := testStore.ListMOObservations(context.Background(), tc.arg)
		tc.result(gotObservations, err)
	}
}

func (ts *MOObservationTestSuite) TestCountMOObservations() {
	t := ts.T()
	timeNow := time.Now()
	n, m := 5, 10
	stations := make([]ObservationsStation, n)
	var selStationIDs []int64
	for i := range stations {
		stations[i] = createRandomStation(t, false)
		for j := 0; j < m; j++ {
			createRandomMOObservation(t, stations[i].ID)
		}
		if i%2 == 0 {
			selStationIDs = append(selStationIDs, stations[i].ID)
		}
	}

	testCases := []struct {
		name   string
		arg    CountMOObservationsParams
		result func(count int64, err error)
	}{
		{
			name: "Default",
			arg: CountMOObservationsParams{
				StationIds: selStationIDs,
			},
			result: func(count int64, err error) {
				require.NoError(t, err)
				require.Equal(t, int64(m*len(selStationIDs)), count)
			},
		},
		{
			name: "WithStartAndEndDate",
			arg: CountMOObservationsParams{
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
			arg: CountMOObservationsParams{
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
			arg: CountMOObservationsParams{
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
		numObservations, err := testStore.CountMOObservations(context.Background(), tc.arg)
		tc.result(numObservations, err)
	}
}

func (ts *MOObservationTestSuite) TestUpdateStationMOObservation() {
	var (
		oldObs  ObservationsMoObservation
		newPres pgtype.Float4
		newRr   pgtype.Float4
	)

	t := ts.T()

	station := createRandomStation(t, false)

	testCases := []struct {
		name        string
		buildArg    func() UpdateStationMOObservationParams
		checkResult func(updatedObs ObservationsMoObservation, err error)
	}{
		{
			name: "SomeValues",
			buildArg: func() UpdateStationMOObservationParams {
				oldObs = createRandomMOObservation(t, station.ID)
				newPres = pgtype.Float4{
					Float32: util.RandomFloat[float32](995.0, 1100.0),
					Valid:   true,
				}
				newRr = pgtype.Float4{
					Float32: util.RandomFloat[float32](0.0, 15.0),
					Valid:   true,
				}

				return UpdateStationMOObservationParams{
					StationID: station.ID,
					ID:        oldObs.ID,
					Pres:      newPres,
					Rr:        newRr,
				}
			},
			checkResult: func(updatedObs ObservationsMoObservation, err error) {
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
			updatedObs, err := testStore.UpdateStationMOObservation(context.Background(), tc.buildArg())
			tc.checkResult(updatedObs, err)
		})
	}
}

func (ts *MOObservationTestSuite) TestDeleteStationMOObservation() {
	t := ts.T()
	station := createRandomStation(t, false)
	obs := createRandomMOObservation(t, station.ID)

	arg := DeleteStationMOObservationParams{
		StationID: obs.StationID,
		ID:        obs.ID,
	}

	err := testStore.DeleteStationMOObservation(context.Background(), arg)
	require.NoError(t, err)

	getArg := GetStationMOObservationParams(arg)
	gotObs, err := testStore.GetStationMOObservation(context.Background(), getArg)
	require.Error(t, err)
	require.Empty(t, gotObs)
}

func createRandomMOObservation(t *testing.T, stationID int64) ObservationsMoObservation {
	arg := CreateStationMOObservationParams{
		Pres: pgtype.Float4{
			Float32: util.RandomFloat[float32](999.0, 1100.9),
			Valid:   true,
		},
		Temp: pgtype.Float4{
			Float32: util.RandomFloat[float32](16.0, 38.0),
			Valid:   true,
		},
		Rr: pgtype.Float4{
			Float32: util.RandomFloat[float32](0.0, 10.0),
			Valid:   true,
		},
		Timestamp: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		StationID: stationID,
	}

	obs, err := testStore.CreateStationMOObservation(context.Background(), arg)
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
