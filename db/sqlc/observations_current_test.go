package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
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
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *CurrentObservationTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
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
