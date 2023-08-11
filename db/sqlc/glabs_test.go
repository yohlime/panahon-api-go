package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GLabsTestSuite struct {
	suite.Suite
}

func TestGLabsTestSuite(t *testing.T) {
	suite.Run(t, new(GLabsTestSuite))
}

func (ts *GLabsTestSuite) SetupTest() {
	util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
}

func (ts *GLabsTestSuite) TearDownTest() {
	runDBMigrationDown(testConfig.MigrationPath, testConfig.DBSource)
}

func (ts *GLabsTestSuite) TestCreateGLabsLoad() {
	t := ts.T()
	station := createRandomStation(t)
	createRandomGlabsLoad(t, station.MobileNumber.Text.String)
}

func createRandomGlabsLoad(t *testing.T, mobileNumber string) GlabsLoad {
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
		MobileNumber: mobileNumber,
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
