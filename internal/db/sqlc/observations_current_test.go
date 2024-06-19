package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/twpayne/go-geom"
)

type CurrentObservationTestSuite struct {
	suite.Suite
}

func TestCurrentObservationTestSuite(t *testing.T) {
	suite.Run(t, new(CurrentObservationTestSuite))
}

func (ts *CurrentObservationTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *CurrentObservationTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *CurrentObservationTestSuite) TestCreateCurrentObservation() {
	createRandomCurrentObservation(ts.T())
}

func (ts *CurrentObservationTestSuite) TestGetLatestStationObservation() {
	t := ts.T()
	n := 10
	stnIdx := util.RandomInt(0, int64(n)-1)
	var station ObservationsStation
	for i := 0; i < n; i++ {
		if i == int(stnIdx) {
			station = createRandomStation(t, true)
			continue
		}
		createRandomStation(t, true)
	}
	obs := createRandomObservation(t, station.ID)

	ctx := context.Background()
	_, err := testStore.InsertCurrentObservations(ctx)
	require.NoError(t, err)

	stnObs, err := testStore.GetLatestStationObservation(ctx, station.ID)
	require.NoError(t, err)
	require.Equal(t, obs.Temp, stnObs.ObservationsCurrent.Temp)
}

func (ts *CurrentObservationTestSuite) TestGetNearestLatestStationObservation() {
	t := ts.T()
	n := 10
	stnIdx := util.RandomInt(0, int64(n)-1)
	var station ObservationsStation
	lat := getRandomLat()
	lon := getRandomLon()
	for i := 0; i < n; i++ {
		if i == int(stnIdx) {
			p := geom.NewPoint(geom.XY).SetSRID(4326).MustSetCoords(geom.Coord{float64(lon), float64(lat)})
			station = createRandomStation(t, util.Point{Point: p})
			continue
		}
		farLon := lon + util.RandomFloat[float32](1, 2)
		farLat := lat + util.RandomFloat[float32](1, 2)
		p := geom.NewPoint(geom.XY).SetSRID(4326).MustSetCoords(geom.Coord{float64(farLon), float64(farLat)})
		createRandomStation(t, util.Point{Point: p})
	}
	obs := createRandomObservation(t, station.ID)

	ctx := context.Background()
	_, err := testStore.InsertCurrentObservations(ctx)
	require.NoError(t, err)

	stnObs, err := testStore.GetNearestLatestStationObservation(
		ctx,
		GetNearestLatestStationObservationParams{
			Lat: station.Lat.Float32,
			Lon: station.Lon.Float32,
		})
	require.NoError(t, err)
	require.Equal(t, obs.Temp, stnObs.ObservationsCurrent.Temp)
}

func createRandomCurrentObservation(t *testing.T) ObservationsCurrent {
	stn := createRandomStation(t, false)
	obs := createRandomObservation(t, stn.ID)
	arg := CreateCurrentObservationParams{
		StationID: obs.StationID,
		Timestamp: obs.Timestamp,
		Temp:      obs.Temp,
		Tn: pgtype.Float4{
			Float32: obs.Temp.Float32 - util.RandomFloat[float32](0.5, 2.0),
			Valid:   true,
		},
		Tx: pgtype.Float4{
			Float32: obs.Temp.Float32 + util.RandomFloat[float32](0.5, 3.0),
			Valid:   true,
		},
		TnTimestamp:   pgtype.Timestamptz{Valid: true},
		TxTimestamp:   pgtype.Timestamptz{Valid: true},
		GustTimestamp: pgtype.Timestamptz{Valid: true},
	}

	cObs, err := testStore.CreateCurrentObservation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, cObs)

	require.Equal(t, arg.StationID, cObs.StationID)

	fDel := 0.01
	require.InDelta(t, arg.Temp.Float32, cObs.Temp.Float32, fDel)
	require.InDelta(t, arg.Tn.Float32, cObs.Tn.Float32, fDel)
	require.InDelta(t, arg.Tx.Float32, cObs.Tx.Float32, fDel)

	tDel := 10 * time.Second
	require.WithinDuration(t, arg.Timestamp.Time, cObs.Timestamp.Time, tDel)
	require.WithinDuration(t, arg.TnTimestamp.Time, cObs.TnTimestamp.Time, tDel)
	require.WithinDuration(t, arg.TxTimestamp.Time, cObs.TxTimestamp.Time, tDel)
	require.WithinDuration(t, arg.GustTimestamp.Time, cObs.GustTimestamp.Time, tDel)

	return cObs
}
