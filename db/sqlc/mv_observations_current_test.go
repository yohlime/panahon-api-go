package db

import (
	"context"
	"testing"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MVCurrentObservationTestSuite struct {
	suite.Suite
}

func TestMVCurrentObservationTestSuite(t *testing.T) {
	suite.Run(t, new(MVCurrentObservationTestSuite))
}

func (ts *MVCurrentObservationTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *MVCurrentObservationTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *MVCurrentObservationTestSuite) TestMVRefreshCurrentObservations() {
	t := ts.T()
	err := testStore.RefreshMVCurrentObservations(context.Background())
	require.NoError(t, err)
}
