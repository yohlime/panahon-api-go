package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestListLufftStationMsg(t *testing.T) {
	station := createRandomStation(t)
	for i := 0; i < 10; i++ {
		createRandomLufftStationMsg(t, station.ID)
	}

	arg := ListLufftStationMsgParams{
		StationID: station.ID,
		Limit:     5,
		Offset:    5,
	}

	healths, err := testStore.ListLufftStationMsg(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, healths, 5)

	for _, h := range healths {
		require.NotEmpty(t, h)
	}
}

func createRandomLufftStationMsg(t *testing.T, stationID int64) ObservationsStationhealth {
	arg := CreateStationHealthParams{
		Message: util.NullString{
			Text: pgtype.Text{
				String: util.RandomString(120),
				Valid:  true,
			},
		},
		Timestamp: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		StationID: stationID,
	}

	health, err := testStore.CreateStationHealth(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, health)

	require.Equal(t, arg.Message, health.Message)
	require.Equal(t, arg.StationID, health.StationID)

	require.True(t, health.UpdatedAt.Time.IsZero())
	require.True(t, health.CreatedAt.Valid)
	require.NotZero(t, health.CreatedAt.Time)

	return health
}
