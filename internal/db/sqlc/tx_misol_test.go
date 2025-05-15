package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MisolStationTxTestSuite struct {
	suite.Suite
}

func TestMisolStationTxTestSuite(t *testing.T) {
	suite.Run(t, new(MisolStationTxTestSuite))
}

func (ts *MisolStationTxTestSuite) SetupTest() {
	err := testMigration.Up()
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *MisolStationTxTestSuite) TearDownTest() {
	err := testMigration.Down()
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *MisolStationTxTestSuite) TestCreateStation() {
	createRandomMisolStationTx(ts.T())
}

func createRandomMisolStationTx(t *testing.T) CreateMisolStationTxResult {
	arg := CreateMisolStationTxParams{
		ID:   util.RandomInt[int64](1, 1000),
		Name: util.RandomString(15),
		Lat: pgtype.Float4{
			Float32: getRandomLon(),
			Valid:   true,
		},
		Lon: pgtype.Float4{
			Float32: getRandomLon(),
			Valid:   true,
		},
		Province: pgtype.Text{
			String: util.RandomString(16),
			Valid:  true,
		},
	}

	mStn, err := testStore.CreateMisolStationTx(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, mStn)

	require.Equal(t, arg.ID, mStn.ID)
	require.Equal(t, arg.Name, mStn.Info.Name)

	return mStn
}
