package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/emiliogozo/panahon-api-go/db/mocks"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListStationObservationsAPI(t *testing.T) {
	n := 10
	stationID := util.RandomInt(1, 100)
	stationObsSlice := make([]db.ObservationsObservation, n)
	for i := 0; i < n; i++ {
		stationObsSlice[i] = randomStationObservation(stationID)
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
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(stationObsSlice, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "InternalError",
			query: Query{
				Page:  1,
				Limit: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsObservation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPage",
			query: Query{
				Page:  -1,
				Limit: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStationObservations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidLimit",
			query: Query{
				Page:  1,
				Limit: 10000,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStationObservations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

			url := fmt.Sprintf("%s/stations/%d/observations", server.config.APIBasePath, stationID)
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

func TestGetStationObservationAPI(t *testing.T) {
	stationID := util.RandomInt(1, 100)
	stationObs := randomStationObservation(stationID)

	testCases := []struct {
		name          string
		params        db.GetStationObservationParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			params: db.GetStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetStationObservationParams{
					ID:        stationObs.ID,
					StationID: stationObs.StationID,
				}
				store.On("GetStationObservation", mock.AnythingOfType("*gin.Context"), arg).
					Return(stationObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stationObs)
			},
		},
		{
			name: "NotFound",
			params: db.GetStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.GetStationObservationParams{
					ID:        stationObs.ID,
					StationID: stationObs.StationID,
				}
				store.On("GetStationObservation", mock.AnythingOfType("*gin.Context"), arg).
					Return(db.ObservationsObservation{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			params: db.GetStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("GetStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsObservation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidID",
			params: db.GetStationObservationParams{
				ID:        0,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

			url := fmt.Sprintf("%s/stations/%d/observations/%d", server.config.APIBasePath, tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestCreateStationObservationAPI(t *testing.T) {
	stationID := util.RandomInt(1, 100)
	stationObs := randomStationObservation(stationID)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"station_id": stationObs.StationID,
				"pres":       stationObs.Pres,
				"temp":       stationObs.Temp,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateStationObservationParams{
					StationID: stationObs.StationID,
					Pres:      stationObs.Pres,
					Temp:      stationObs.Temp,
				}

				store.On("CreateStationObservation", mock.AnythingOfType("*gin.Context"), arg).
					Return(stationObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stationObs)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"station_id": stationObs.StationID,
				"pres":       stationObs.Pres,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("CreateStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsObservation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/stations/%d/observations", server.config.APIBasePath, stationObs.StationID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestUpdateStationObservationAPI(t *testing.T) {
	stationID := util.RandomInt(1, 100)
	stationObs := randomStationObservation(stationID)

	testCases := []struct {
		name          string
		params        db.UpdateStationObservationParams
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			params: db.UpdateStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			body: gin.H{
				"id":         stationObs.ID,
				"station_id": stationObs.StationID,
				"pres":       stationObs.Pres,
				"temp":       stationObs.Temp,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateStationObservationParams{
					ID:        stationObs.ID,
					StationID: stationObs.StationID,
					Pres:      stationObs.Pres,
					Temp:      stationObs.Temp,
				}

				store.On("UpdateStationObservation", mock.AnythingOfType("*gin.Context"), arg).
					Return(stationObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stationObs)
			},
		},
		{
			name: "InternalError",
			params: db.UpdateStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("UpdateStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsObservation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "NotFound",
			params: db.UpdateStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("UpdateStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsObservation{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
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

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/stations/%d/observations/%d", server.config.APIBasePath, tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteStationObservationObservationAPI(t *testing.T) {
	stationID := util.RandomInt(1, 100)
	stationObs := randomStationObservation(stationID)

	testCases := []struct {
		name          string
		params        db.UpdateStationObservationParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			params: db.UpdateStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("DeleteStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name: "InternalError",
			params: db.UpdateStationObservationParams{
				ID:        stationObs.ID,
				StationID: stationObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.On("DeleteStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			url := fmt.Sprintf("%s/stations/%d/observations/%d", server.config.APIBasePath, tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomStationObservation(stationID int64) db.ObservationsObservation {
	return db.ObservationsObservation{
		ID:        util.RandomInt(1, 1000),
		StationID: stationID,
		Pres:      util.NullFloat4{Float4: pgtype.Float4{Float32: util.RandomFloat(900.0, 1000.0), Valid: true}},
		Temp:      util.NullFloat4{Float4: pgtype.Float4{Float32: util.RandomFloat(25.0, 35.0), Valid: true}},
	}
}

func requireBodyMatchStationObservation(t *testing.T, body *bytes.Buffer, stationObs db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObs db.ObservationsObservation
	err = json.Unmarshal(data, &gotStationObs)
	require.NoError(t, err)
	require.Equal(t, stationObs, gotStationObs)
}

func requireBodyMatchStationObservations(t *testing.T, body *bytes.Buffer, stationObsSlice []db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObsSlice []db.ObservationsObservation
	err = json.Unmarshal(data, &gotStationObsSlice)
	require.NoError(t, err)
	require.Equal(t, stationObsSlice, gotStationObsSlice)
}
