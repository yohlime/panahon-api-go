package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateStation(t *testing.T) {
	createRandomStation(t)
}

func TestGetStation(t *testing.T) {
	station := createRandomStation(t)

	gotStation, err := testStore.GetStation(context.Background(), station.ID)

	require.NoError(t, err)
	require.NotEmpty(t, gotStation)

	require.Equal(t, station, gotStation)
}

func TestListStations(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomStation(t)
	}

	arg := ListStationsParams{
		Limit:  5,
		Offset: 5,
	}
	stations, err := testStore.ListStations(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, stations, 5)

	for _, station := range stations {
		require.NotEmpty(t, station)
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
		})
	}
}

func TestDeleteStation(t *testing.T) {
	station := createRandomStation(t)

	err := testStore.DeleteStation(context.Background(), station.ID)
	require.NoError(t, err)

	gotStation, err := testStore.GetStation(context.Background(), station.ID)
	require.Error(t, err)
	require.Empty(t, gotStation)
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
