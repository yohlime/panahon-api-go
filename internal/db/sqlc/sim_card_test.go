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

type SimCardTestSuite struct {
	suite.Suite
}

func TestSimCardTestSuite(t *testing.T) {
	suite.Run(t, new(SimCardTestSuite))
}

func (ts *SimCardTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *SimCardTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *SimCardTestSuite) TestCreateSimCard() {
	createRandomSimCard(ts.T())
}

func createRandomSimCard(t *testing.T) SimCard {
	simCard := randomSimCard()
	arg := CreateSimCardParams{
		MobileNumber: simCard.MobileNumber,
		Type:         simCard.Type,
	}

	simCard, err := testStore.CreateSimCard(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, simCard)

	require.Equal(t, arg.MobileNumber, simCard.MobileNumber)
	require.Equal(t, arg.Type, simCard.Type)
	require.True(t, simCard.UpdatedAt.Time.IsZero())
	require.True(t, simCard.CreatedAt.Valid)
	require.NotZero(t, simCard.CreatedAt.Time)

	return simCard
}

func randomSimCard() SimCard {
	return SimCard{
		MobileNumber: util.RandomMobileNumber(),
		Type: pgtype.Text{
			String: util.RandomString(6),
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	}
}
