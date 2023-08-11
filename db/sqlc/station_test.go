package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateStation(t *testing.T) {
	station := createRandomStation(t)

	// cleanup
	_deleteStation(t, station.ID)
}

func TestGetStation(t *testing.T) {
	station := createRandomStation(t)

	gotStation, err := testStore.GetStation(context.Background(), station.ID)

	require.NoError(t, err)
	require.NotEmpty(t, gotStation)

	require.Equal(t, station, gotStation)

	// cleanup
	_deleteStation(t, station.ID)
}

func TestListStations(t *testing.T) {
	n := 10
	stations := make([]ObservationsStation, n)
	for i := 0; i < n; i++ {
		stations[i] = createRandomStation(t)
	}

	arg := ListStationsParams{
		Limit:  5,
		Offset: 5,
	}
	gotStations, err := testStore.ListStations(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, gotStations, 5)

	for _, station := range gotStations {
		require.NotEmpty(t, station)
	}

	// cleanup
	for _, station := range stations {
		_deleteStation(t, station.ID)
	}
}

func TestCountStations(t *testing.T) {
	n := 10
	stations := make([]ObservationsStation, n)
	for i := 0; i < n; i++ {
		stations[i] = createRandomStation(t)
	}

	numStations, err := testStore.CountStations(context.Background())
	require.NoError(t, err)
	require.Equal(t, numStations, int64(n))

	// cleanup
	for _, station := range stations {
		_deleteStation(t, station.ID)
	}
}

func TestUpdateStation(t *testing.T) {
	var (
		oldStation      ObservationsStation
		newName         util.NullString
		newMobileNumber util.NullString
		newLat          util.NullFloat4
		newLon          util.NullFloat4
	)

	testCases := []struct {
		name        string
		buildArg    func() UpdateStationParams
		checkResult func(updatedStation ObservationsStation, err error)
	}{
		{
			name: "NameOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t)
				newName = util.NullString{
					Text: pgtype.Text{
						String: util.RandomString(12),
						Valid:  true,
					},
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
				require.Equal(t, oldStation.MobileNumber, updatedStation.MobileNumber)
			},
		},
		{
			name: "MobileNumberOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t)
				newMobileNumber = util.NullString{
					Text: pgtype.Text{
						String: util.RandomMobileNumber(),
						Valid:  true,
					},
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
			},
		},
		{
			name: "LatLonOnly",
			buildArg: func() UpdateStationParams {
				oldStation = createRandomStation(t)
				newLat = util.RandomNullFloat4(-90.0, 90.0)
				newLon = util.RandomNullFloat4(0.0, 359.9)

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

			// cleanup
			_deleteStation(t, updatedStation.ID)
		})
	}
}

func TestDeleteStation(t *testing.T) {
	station := createRandomStation(t)

	// cleanup
	_deleteStation(t, station.ID)
}

func createRandomStation(t *testing.T) ObservationsStation {
	mobileNum := util.RandomMobileNumber()

	arg := CreateStationParams{
		Name: util.RandomString(16),
		MobileNumber: util.NullString{
			Text: pgtype.Text{
				String: mobileNum,
				Valid:  true,
			},
		},
	}
	station, err := testStore.CreateStation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, station)

	require.Equal(t, arg.Name, station.Name)
	require.Equal(t, arg.MobileNumber, station.MobileNumber)
	require.True(t, station.UpdatedAt.Time.IsZero())
	require.True(t, station.CreatedAt.Valid)
	require.NotZero(t, station.CreatedAt.Time)

	return station
}

func _deleteStation(t *testing.T, stationID int64) {
	err := testStore.DeleteStation(context.Background(), stationID)
	require.NoError(t, err)

	gotStation, err := testStore.GetStation(context.Background(), stationID)
	require.Error(t, err)
	require.Empty(t, gotStation)
}
