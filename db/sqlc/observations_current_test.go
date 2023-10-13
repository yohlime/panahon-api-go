package db

import (
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CurrentObservationTestSuite struct {
	suite.Suite
}

func TestCurrentObservationTestSuite(t *testing.T) {
	suite.Run(t, new(CurrentObservationTestSuite))
}

func (ts *CurrentObservationTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *CurrentObservationTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}
