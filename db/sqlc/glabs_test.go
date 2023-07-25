package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateGLabsLoad(t *testing.T) {
	createRandomGlabsLoad(t)
}

func createRandomGlabsLoad(t *testing.T) GlabsLoad {
	station := createRandomStation(t)
	arg := CreateGLabsLoadParams{
		Promo: util.NullString{
			Text: pgtype.Text{
				String: util.RandomString(10),
				Valid:  true,
			},
		},
		TransactionID: util.RandomNullInt4(1000000, 9999999),
		Status: util.NullString{
			Text: pgtype.Text{
				String: util.RandomString(10),
				Valid:  true,
			},
		},
		MobileNumber: station.MobileNumber.Text.String,
	}

	gLabsLoad, err := testStore.CreateGLabsLoad(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, gLabsLoad)

	require.Equal(t, arg.Promo, gLabsLoad.Promo)
	require.Equal(t, arg.TransactionID, gLabsLoad.TransactionID)
	require.Equal(t, arg.Status, gLabsLoad.Status)
	require.Equal(t, arg.MobileNumber, gLabsLoad.MobileNumber)
	require.NotEmpty(t, gLabsLoad.ID)
	require.True(t, gLabsLoad.UpdatedAt.Time.IsZero())
	require.True(t, gLabsLoad.CreatedAt.Valid)
	require.NotZero(t, gLabsLoad.CreatedAt.Time)

	return gLabsLoad
}
