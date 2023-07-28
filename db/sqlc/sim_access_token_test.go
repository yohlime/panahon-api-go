package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateSimAccessToken(t *testing.T) {
	simCard := createRandomSimCard(t)
	createRandomSimAccessToken(t, simCard.MobileNumber)
}

func TestGetSimAccessToken(t *testing.T) {
	simCard := createRandomSimCard(t)
	accTkn := createRandomSimAccessToken(t, simCard.MobileNumber)
	gotAccTkn, err := testStore.GetSimAccessToken(context.Background(), accTkn.AccessToken)
	require.NoError(t, err)
	require.NotEmpty(t, gotAccTkn)

	require.Equal(t, accTkn.AccessToken, gotAccTkn.AccessToken)
	require.Equal(t, accTkn.Type, gotAccTkn.Type)
	require.Equal(t, accTkn.MobileNumber, gotAccTkn.MobileNumber)
	require.WithinDuration(t, accTkn.CreatedAt.Time, gotAccTkn.CreatedAt.Time, time.Second)
	require.WithinDuration(t, accTkn.UpdatedAt.Time, gotAccTkn.UpdatedAt.Time, time.Second)
}

func TestDeleteSimAccessToken(t *testing.T) {
	simCard := createRandomSimCard(t)
	accTkn := createRandomSimAccessToken(t, simCard.MobileNumber)

	err := testStore.DeleteSimAccessToken(context.Background(), accTkn.AccessToken)
	require.NoError(t, err)
}

func createRandomSimAccessToken(t *testing.T, mobileNumber string) SimAccessToken {
	simAccessToken := randomSimAccessToken(mobileNumber)
	arg := CreateSimAccessTokenParams{
		AccessToken:  simAccessToken.AccessToken,
		Type:         simAccessToken.Type,
		MobileNumber: mobileNumber,
	}

	accTkn, err := testStore.CreateSimAccessToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, accTkn)

	require.Equal(t, arg.AccessToken, accTkn.AccessToken)
	require.Equal(t, arg.Type, accTkn.Type)
	require.Equal(t, arg.MobileNumber, accTkn.MobileNumber)
	require.True(t, accTkn.UpdatedAt.Time.IsZero())
	require.True(t, accTkn.CreatedAt.Valid)
	require.NotZero(t, accTkn.CreatedAt.Time)

	return accTkn
}

func randomSimAccessToken(mobileNumber string) SimAccessToken {
	return SimAccessToken{
		AccessToken:  util.RandomString(32),
		Type:         util.RandomString(6),
		MobileNumber: mobileNumber,
		CreatedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	}
}
