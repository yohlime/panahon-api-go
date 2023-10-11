package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FOCSimAccessTokenTxTestSuite struct {
	suite.Suite
}

func TestFOCSimAccessTokenTxTestSuite(t *testing.T) {
	suite.Run(t, new(FOCSimAccessTokenTxTestSuite))
}

func (ts *FOCSimAccessTokenTxTestSuite) SetupTest() {
	err := util.RunDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "db migration problem")
}

func (ts *FOCSimAccessTokenTxTestSuite) TearDownTest() {
	err := util.ReverseDBMigration(testConfig.MigrationPath, testConfig.DBSource)
	require.NoError(ts.T(), err, "reverse db migration problem")
}

func (ts *FOCSimAccessTokenTxTestSuite) TestFirstOrCreateSimAccessToken() {
	var newAccTkn SimAccessToken

	t := ts.T()
	simCard := createRandomSimCard(t)
	accTkn := createRandomSimAccessToken(t, simCard.MobileNumber)

	newSimCard := randomSimCard()

	testCases := []struct {
		name        string
		buildArg    func() FirstOrCreateSimAccessTokenTxParams
		checkResult func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error)
	}{
		{
			name: "AccessTokenExists",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      accTkn.AccessToken,
					AccessTokenType:  accTkn.Type,
					MobileNumber:     simCard.MobileNumber,
					MobileNumberType: simCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, gotAccTkn)
				require.False(t, gotAccTkn.IsCreated)

				require.Equal(t, accTkn.AccessToken, gotAccTkn.AccessToken.AccessToken)
				require.Equal(t, accTkn.Type, gotAccTkn.AccessToken.Type)
				require.Equal(t, accTkn.MobileNumber, gotAccTkn.AccessToken.MobileNumber)
				require.WithinDuration(t, accTkn.CreatedAt.Time, gotAccTkn.AccessToken.CreatedAt.Time, time.Second)
				require.WithinDuration(t, accTkn.UpdatedAt.Time, gotAccTkn.AccessToken.UpdatedAt.Time, time.Second)
			},
		},
		{
			name: "NewAccessToken",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				newAccTkn = randomSimAccessToken(simCard.MobileNumber)
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      newAccTkn.AccessToken,
					AccessTokenType:  newAccTkn.Type,
					MobileNumber:     simCard.MobileNumber,
					MobileNumberType: simCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, gotAccTkn)
				require.True(t, gotAccTkn.IsCreated)

				require.Equal(t, newAccTkn.AccessToken, gotAccTkn.AccessToken.AccessToken)
				require.Equal(t, newAccTkn.Type, gotAccTkn.AccessToken.Type)
				require.Equal(t, newAccTkn.MobileNumber, gotAccTkn.AccessToken.MobileNumber)
			},
		},
		{
			name: "NewMobileNumberAndAccessToken",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				newAccTkn = randomSimAccessToken(newSimCard.MobileNumber)
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      newAccTkn.AccessToken,
					AccessTokenType:  newAccTkn.Type,
					MobileNumber:     newSimCard.MobileNumber,
					MobileNumberType: newSimCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, gotAccTkn)
				require.True(t, gotAccTkn.IsCreated)

				require.Equal(t, newAccTkn.AccessToken, gotAccTkn.AccessToken.AccessToken)
				require.Equal(t, newAccTkn.Type, gotAccTkn.AccessToken.Type)
				require.Equal(t, newAccTkn.MobileNumber, gotAccTkn.AccessToken.MobileNumber)
			},
		},
		{
			name: "MissingAccessTokenParam",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      "",
					AccessTokenType:  accTkn.Type,
					MobileNumber:     simCard.MobileNumber,
					MobileNumberType: simCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.Error(t, err)
				require.Empty(t, gotAccTkn)
			},
		},
		{
			name: "MissingMobileNumberParam",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      accTkn.AccessToken,
					AccessTokenType:  accTkn.Type,
					MobileNumber:     "",
					MobileNumberType: simCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.Error(t, err)
				require.Empty(t, gotAccTkn)
			},
		},
		{
			name: "MissingAccessTokenTypeParam",
			buildArg: func() FirstOrCreateSimAccessTokenTxParams {
				return FirstOrCreateSimAccessTokenTxParams{
					AccessToken:      accTkn.AccessToken,
					AccessTokenType:  "",
					MobileNumber:     simCard.MobileNumber,
					MobileNumberType: simCard.Type,
				}
			},
			checkResult: func(gotAccTkn FirstOrCreateSimAccessTokenTxResult, err error) {
				require.Error(t, err)
				require.Empty(t, gotAccTkn)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			gotAccTkn, err := testStore.FirstOrCreateSimAccessTokenTx(context.Background(), tc.buildArg())

			tc.checkResult(gotAccTkn, err)
		})
	}
}
