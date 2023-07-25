package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/emiliogozo/panahon-api-go/db/mocks"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLufftMsgLog(t *testing.T) {
	n := 10
	stationID := util.RandomInt(1, 100)
	lufftStationMsgs := make([]db.ListLufftStationMsgRow, n)
	for i := 0; i < n; i++ {
		msgSlice := randomLufftMsgLog(stationID)
		lufftStationMsgs[i] = db.ListLufftStationMsgRow{
			Message:   msgSlice.Message,
			Timestamp: msgSlice.Timestamp,
		}
	}

	type Query struct {
		Page  int32
		Limit int32
	}

	testCases := []struct {
		name          string
		query         Query
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			query: Query{
				Page:  1,
				Limit: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListLufftStationMsg(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(lufftStationMsgs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchLufftMsgLogs(t, recorder.Body, lufftStationMsgs)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/lufft/%d/logs", server.config.APIBasePath, stationID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to request URL
			q := request.URL.Query()
			q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			q.Add("limit", fmt.Sprintf("%d", tc.query.Limit))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomLufftMsgLog(stationID int64) db.ObservationsStationhealth {
	return db.ObservationsStationhealth{
		ID:        util.RandomInt(1, 1000),
		StationID: stationID,
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
	}
}

func requireBodyMatchLufftMsgLogs(t *testing.T, body *bytes.Buffer, lufftStationMsgs []db.ListLufftStationMsgRow) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotLufftStationMsgs []db.ListLufftStationMsgRow
	err = json.Unmarshal(data, &gotLufftStationMsgs)
	require.NoError(t, err)
	for m, msg := range lufftStationMsgs {
		require.Equal(t, msg.Message, gotLufftStationMsgs[m].Message)
		require.WithinDuration(t, msg.Timestamp.Time, gotLufftStationMsgs[m].Timestamp.Time, time.Second)
	}

}
