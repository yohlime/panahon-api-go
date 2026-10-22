package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetStationStatsAPI(t *testing.T) {
	testCases := []struct {
		name         string
		query        string
		buildStubs   func(store *mockdb.MockStore)
		expectedCode int
		expected     models.StationStats
	}{
		{
			name:  "AllStations",
			query: "",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetStationStats(mock.AnythingOfType("*gin.Context")).
					Return(db.GetStationStatsRow{Total: 10, Online: 7, Offline: 3, Reporting: 6}, nil)
			},
			expectedCode: http.StatusOK,
			expected:     models.StationStats{Total: 10, Online: 7, Offline: 3, Reporting: 6},
		},
		{
			name:  "AccessibleWithoutAuth",
			query: "?accessible=true",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetStationStatsByIds(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(ids []int64) bool {
						return ids != nil && len(ids) == 0
					})).
					Return(db.GetStationStatsByIdsRow{}, nil)
			},
			expectedCode: http.StatusOK,
			expected:     models.StationStats{},
		},
		{
			name:  "StoreError",
			query: "",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetStationStats(mock.AnythingOfType("*gin.Context")).
					Return(db.GetStationStatsRow{}, sql.ErrConnDone)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			handler := newTestHandler(store, nil)
			router := gin.Default()
			router.GET("/stats", handler.GetStationStats)

			request, err := http.NewRequest(http.MethodGet, "/stats"+tc.query, nil)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			store.AssertExpectations(t)
			require.Equal(t, tc.expectedCode, recorder.Code)

			if tc.expectedCode == http.StatusOK {
				var got models.StationStats
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
				require.Equal(t, tc.expected, got)
			}
		})
	}
}
