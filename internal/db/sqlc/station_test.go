package db

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/twpayne/go-geom"
)

type StationTestSuite struct {
	suite.Suite
}

func TestStationTestSuite(t *testing.T) {
	suite.Run(t, new(StationTestSuite))
}

func (ts *StationTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *StationTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *StationTestSuite) TestCreateStation() {
	createRandomStation(ts.T(), true)
}

func (ts *StationTestSuite) TestGetStation() {
	t := ts.T()
	station := createRandomStation(t, false)

	gotStation, err := testStore.GetStation(context.Background(), station.ID)

	require.NoError(t, err)
	require.NotEmpty(t, gotStation)

	require.Equal(t, gotStation.Name, station.Name)
	require.Equal(t, gotStation.MobileNumber, station.MobileNumber)
}

func (ts *StationTestSuite) TestListStations() {
	t := ts.T()
	n := 10
	ctx := context.Background()
	for i := 0; i < n; i++ {
		station := createRandomStation(t, false)
		if i%3 == 0 {
			testStore.UpdateStation(ctx,
				UpdateStationParams{
					ID:     station.ID,
					Status: pgtype.Text{String: "ONLINE", Valid: true},
				},
			)
		} else {
			testStore.UpdateStation(ctx,
				UpdateStationParams{
					ID:     station.ID,
					Status: pgtype.Text{String: "OFFLINE", Valid: true},
				},
			)
		}
	}

	arg := ListStationsParams{
		Limit:  pgtype.Int4{Int32: 5, Valid: true},
		Offset: 5,
	}
	gotStations, err := testStore.ListStations(ctx, arg)
	require.NoError(t, err)
	require.Len(t, gotStations, 5)

	for _, station := range gotStations {
		require.NotEmpty(t, station)
	}

	arg = ListStationsParams{
		Status: pgtype.Text{String: "ONLINE", Valid: true},
	}
	gotStations, err = testStore.ListStations(ctx, arg)
	require.NoError(t, err)
	require.Len(t, gotStations, 4)

	for _, station := range gotStations {
		require.NotEmpty(t, station)
	}
}

func (ts *StationTestSuite) TestListStationsWithinRadius() {
	t := ts.T()
	cLat := getRandomLat()
	cLon := getRandomLon()
	cR := float32(1.0)
	n := 10
	for i := 0; i < n; i++ {
		var lon, lat float32
		if i%2 == 0 {
			lon, lat = getRandomCoordsFromCircle(getRandomCoordsFromCircleParams{
				cX:        cLon,
				cY:        cLat,
				r:         cR,
				minOffset: 2.0,
				maxOffset: 3.0,
			})
		} else {
			lon, lat = getRandomCoordsFromCircle(getRandomCoordsFromCircleParams{
				cX: cLon,
				cY: cLat,
				r:  cR,
			})
		}
		p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{float64(lon), float64(lat)}).SetSRID(4326)
		createRandomStation(t, util.Point{Point: p})
	}

	arg := ListStationsWithinRadiusParams{
		Cx:     cLon,
		Cy:     cLat,
		R:      cR,
		Limit:  pgtype.Int4{Int32: int32(n), Valid: true},
		Offset: 0,
	}
	gotStations, err := testStore.ListStationsWithinRadius(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotStations, 5)

	for i := range gotStations {
		require.NotEmpty(t, gotStations[i])
	}
}

func (ts *StationTestSuite) TestListStationsWithinBBox() {
	t := ts.T()
	xMin, yMin, xMax, yMax := 120.0, 5.0, 122.0, 6.0
	n := 10
	for i := 0; i < n; i++ {
		var lat, lon float32
		if i%2 == 0 {
			lon = util.RandomFloat(float32(xMin), float32(xMax))
			lat = util.RandomFloat(float32(yMin), float32(yMax))
		} else {
			lon = util.RandomFloat(float32(xMax), float32(xMax+1.0))
			lat = util.RandomFloat(float32(yMax), float32(yMax+1.0))
		}
		p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{float64(lon), float64(lat)}).SetSRID(4326)
		createRandomStation(t, util.Point{Point: p})
	}

	arg := ListStationsWithinBBoxParams{
		Xmin:   float32(xMin),
		Ymin:   float32(yMin),
		Xmax:   float32(xMax),
		Ymax:   float32(yMax),
		Limit:  pgtype.Int4{Int32: int32(n), Valid: true},
		Offset: 0,
	}
	gotStations, err := testStore.ListStationsWithinBBox(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotStations, 5)

	for _, station := range gotStations {
		require.NotEmpty(t, station)
	}
}

func (ts *StationTestSuite) TestCountStations() {
	t := ts.T()
	n := 10
	ctx := context.Background()
	for i := 0; i < n; i++ {
		station := createRandomStation(t, false)
		if i%3 == 0 {
			testStore.UpdateStation(ctx,
				UpdateStationParams{
					ID:     station.ID,
					Status: pgtype.Text{String: "ONLINE", Valid: true},
				},
			)
		} else {
			testStore.UpdateStation(ctx,
				UpdateStationParams{
					ID:     station.ID,
					Status: pgtype.Text{String: "OFFLINE", Valid: true},
				},
			)
		}
	}

	numStations, err := testStore.CountStations(ctx, pgtype.Text{})
	require.NoError(t, err)
	require.Equal(t, int64(n), numStations)

	numStations, err = testStore.CountStations(ctx, pgtype.Text{String: "ONLINE", Valid: true})
	require.NoError(t, err)
	require.Equal(t, int64(4), numStations)
}

func (ts *StationTestSuite) TestCountStationsWithinRadius() {
	t := ts.T()
	cLat := getRandomLat()
	cLon := getRandomLon()
	cR := float32(1.0)
	n := 10
	for i := 0; i < n; i++ {
		var lon, lat float32
		if i%2 == 0 {
			lon, lat = getRandomCoordsFromCircle(getRandomCoordsFromCircleParams{
				cX:        cLon,
				cY:        cLat,
				r:         cR,
				minOffset: 2.0,
				maxOffset: 3.0,
			})
		} else {
			lon, lat = getRandomCoordsFromCircle(getRandomCoordsFromCircleParams{
				cX: cLon,
				cY: cLat,
				r:  cR,
			})
		}
		p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{float64(lon), float64(lat)}).SetSRID(4326)
		createRandomStation(t, util.Point{Point: p})
	}
	arg := CountStationsWithinRadiusParams{
		Cx: cLon,
		Cy: cLat,
		R:  cR,
	}
	numStations, err := testStore.CountStationsWithinRadius(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, numStations, int64(5))
}

func (ts *StationTestSuite) TestCountStationsWithinBBox() {
	t := ts.T()
	xMin, yMin, xMax, yMax := 120.0, 5.0, 122.0, 6.0
	n := 10
	for i := 0; i < n; i++ {
		var lat, lon float32
		if i%2 == 0 {
			lon = util.RandomFloat(float32(xMin), float32(xMax))
			lat = util.RandomFloat(float32(yMin), float32(yMax))
		} else {
			lon = util.RandomFloat(float32(xMax), float32(xMax+1.0))
			lat = util.RandomFloat(float32(yMax), float32(yMax+1.0))
		}
		p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{float64(lon), float64(lat)}).SetSRID(4326)
		createRandomStation(t, util.Point{Point: p})
	}

	arg := CountStationsWithinBBoxParams{
		Xmin: float32(xMin),
		Ymin: float32(yMin),
		Xmax: float32(xMax),
		Ymax: float32(yMax),
	}
	numStations, err := testStore.CountStationsWithinBBox(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, numStations, int64(5))
}

func (ts *StationTestSuite) TestUpdateStation() {
	var (
		oldStation      ObservationsStation
		newName         pgtype.Text
		newMobileNumber pgtype.Text
		newLat          pgtype.Float4
		newLon          pgtype.Float4
	)

	t := ts.T()

	testCases := []struct {
		name        string
		buildArg    func() UpdateStationParams
		checkResult func(updatedStation ObservationsStation, err error)
	}{
		{
			name: "NameOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t, true)
				newName = pgtype.Text{
					String: util.RandomString(12),
					Valid:  true,
				}

				return UpdateStationParams{
					ID:   oldStation.ID,
					Name: newName,
				}
			},
			checkResult: func(updatedStation ObservationsStation, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldStation.Name, updatedStation.Name)
				require.Equal(t, oldStation.Lat, updatedStation.Lat)
				require.Equal(t, oldStation.Lon, updatedStation.Lon)
				require.Equal(t, oldStation.Geom, updatedStation.Geom)
				require.Equal(t, oldStation.MobileNumber, updatedStation.MobileNumber)
			},
		},
		{
			name: "MobileNumberOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t, true)
				newMobileNumber = pgtype.Text{
					String: util.RandomMobileNumber(),
					Valid:  true,
				}
				return UpdateStationParams{
					ID:           oldStation.ID,
					MobileNumber: newMobileNumber,
				}
			},
			checkResult: func(updatedStation ObservationsStation, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldStation.MobileNumber, updatedStation.MobileNumber)
				require.Equal(t, oldStation.Name, updatedStation.Name)
				require.Equal(t, oldStation.Lat, updatedStation.Lat)
				require.Equal(t, oldStation.Lon, updatedStation.Lon)
				require.Equal(t, oldStation.Geom, updatedStation.Geom)
			},
		},
		{
			name: "LatLonOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t, true)
				newLat = pgtype.Float4{
					Float32: getRandomLat(),
					Valid:   true,
				}
				newLon = pgtype.Float4{
					Float32: getRandomLon(),
					Valid:   true,
				}

				return UpdateStationParams{
					ID:  oldStation.ID,
					Lat: newLat,
					Lon: newLon,
				}
			},
			checkResult: func(updatedStation ObservationsStation, err error) {
				require.NoError(t, err)
				require.NotEqual(t, oldStation.Lat, updatedStation.Lat)
				require.NotEqual(t, oldStation.Lon, updatedStation.Lon)
				require.NotEqual(t, oldStation.Geom, updatedStation.Geom)
				require.Equal(t, oldStation.Name, updatedStation.Name)
				require.Equal(t, oldStation.MobileNumber, updatedStation.MobileNumber)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			updatedStation, err := testStore.UpdateStation(context.Background(), tc.buildArg())
			tc.checkResult(updatedStation, err)
		})
	}
}

func (ts *StationTestSuite) TestDeleteStation() {
	t := ts.T()
	station := createRandomStation(t, false)

	err := testStore.DeleteStation(context.Background(), station.ID)
	require.NoError(t, err)

	gotStation, err := testStore.GetStation(context.Background(), station.ID)
	require.Error(t, err)
	require.Empty(t, gotStation)
}

func createRandomStation(t *testing.T, geom any) ObservationsStation {
	mobileNum := util.RandomMobileNumber()

	rMon := -util.RandomInt(0, 2)
	rDay := -util.RandomInt(1, 20)
	timeNow := time.Now().AddDate(0, rMon, rDay)
	dateInstalled := pgtype.Date{
		Time:  time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}

	arg := CreateStationParams{
		Name: util.RandomString(16),
		MobileNumber: pgtype.Text{
			String: mobileNum,
			Valid:  true,
		},
		DateInstalled: dateInstalled,
	}

	switch g := geom.(type) {
	case bool:
		if g {
			arg.Lat = pgtype.Float4{
				Float32: getRandomLat(),
				Valid:   true,
			}
			arg.Lon = pgtype.Float4{
				Float32: getRandomLon(),
				Valid:   true,
			}
		}
	case util.Point:
		arg.Lon = pgtype.Float4{
			Float32: float32(g.X()),
			Valid:   true,
		}
		arg.Lat = pgtype.Float4{
			Float32: float32(g.Y()),
			Valid:   true,
		}
	}

	station, err := testStore.CreateStation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, station)

	require.Equal(t, arg.Name, station.Name)
	require.Equal(t, arg.MobileNumber, station.MobileNumber)
	require.WithinDuration(t, dateInstalled.Time, station.DateInstalled.Time, time.Second*10)
	require.True(t, station.UpdatedAt.Time.IsZero())
	require.True(t, station.CreatedAt.Valid)
	require.NotZero(t, station.CreatedAt.Time)

	return station
}

func getRandomLat() float32 {
	return util.RandomFloat[float32](5.5, 18.6)
}

func getRandomLon() float32 {
	return util.RandomFloat[float32](117.15, 126.6)
}

type getRandomCoordsFromCircleParams struct {
	cX, cY, r            float32
	minOffset, maxOffset float32
}

func getRandomCoordsFromCircle(arg getRandomCoordsFromCircleParams) (lon, lat float32) {
	theta := 2 * math.Pi * float64(util.RandomFloat(0.0, 1.0))

	minOffset, maxOffset := float32(0.0), float32(1.0)
	if (arg.minOffset > 0) && (arg.maxOffset > arg.minOffset) {
		minOffset = arg.minOffset
		maxOffset = arg.maxOffset
	}

	d := arg.r * float32(math.Sqrt(float64(util.RandomFloat(minOffset, maxOffset))))
	lon = arg.cX + d*float32(math.Cos(theta))
	lat = arg.cY + d*float32(math.Sin(theta))
	return
}
