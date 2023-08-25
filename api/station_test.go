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

func TestListStationsAPI(t *testing.T) {
	n := 10
	stations := make([]db.ObservationsStation, n)
	for i := 0; i < n; i++ {
		stations[i] = randomStation()
	}

	type Query struct {
		Page    int32
		PerPage int32
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
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(stations, nil)
				store.EXPECT().CountStations(mock.AnythingOfType("*gin.Context")).Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStations(t, recorder.Body, stations)
			},
		},
		{
			name: "InternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsStation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPage",
			query: Query{
				Page:    -1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidLimit",
			query: Query{
				Page:    1,
				PerPage: 10000,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "EmptySlice",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsStation{}, nil)
				store.EXPECT().CountStations(mock.AnythingOfType("*gin.Context")).Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStations(t, recorder.Body, []db.ObservationsStation{})
			},
		},
		{
			name: "CountInternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsStation{}, nil)
				store.EXPECT().CountStations(mock.AnythingOfType("*gin.Context")).Return(0, sql.ErrConnDone)
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

			url := fmt.Sprintf("%s/stations", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to request URL
			q := request.URL.Query()
			q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			q.Add("per_page", fmt.Sprintf("%d", tc.query.PerPage))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetStationAPI(t *testing.T) {
	station := randomStation()

	testCases := []struct {
		name          string
		stationID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:      "OK",
			stationID: station.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), station.ID).
					Return(station, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStation(t, recorder.Body, station)
			},
		},
		{
			name:      "NotFound",
			stationID: station.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), station.ID).
					Return(db.ObservationsStation{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			stationID: station.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), station.ID).
					Return(db.ObservationsStation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			stationID: 0,
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetStation", mock.AnythingOfType("*gin.Context"), mock.Anything)
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

			url := fmt.Sprintf("%s/stations/%d", server.config.APIBasePath, tc.stationID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestCreateStationAPI(t *testing.T) {
	station := randomStation()

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"name":     station.Name,
				"lat":      station.Lat,
				"lon":      station.Lon,
				"province": station.Province,
				"region":   station.Region,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateStationParams{
					Name:     station.Name,
					Lat:      station.Lat,
					Lon:      station.Lon,
					Province: station.Province,
					Region:   station.Region,
				}

				store.EXPECT().CreateStation(mock.AnythingOfType("*gin.Context"), arg).
					Return(station, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchStation(t, recorder.Body, station)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"name": station.Name,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateStation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsStation{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("%s/stations", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestUpdateStationAPI(t *testing.T) {
	station := randomStation()

	testCases := []struct {
		name          string
		stationID     int64
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:      "OK",
			stationID: station.ID,
			body: gin.H{
				"id":       station.ID,
				"name":     station.Name,
				"lat":      station.Lat,
				"lon":      station.Lon,
				"province": station.Province,
				"region":   station.Region,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateStationParams{
					ID: station.ID,
					Name: util.NullString{
						Text: pgtype.Text{
							String: station.Name,
							Valid:  true,
						}},
					Lat:      station.Lat,
					Lon:      station.Lon,
					Province: station.Province,
					Region:   station.Region,
				}

				store.EXPECT().UpdateStation(mock.AnythingOfType("*gin.Context"), arg).
					Return(station, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStation(t, recorder.Body, station)
			},
		},
		{
			name:      "InternalError",
			stationID: station.ID,
			body:      gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateStation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsStation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "StationNotFound",
			stationID: station.ID,
			body:      gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateStation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsStation{}, db.ErrRecordNotFound)
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

			url := fmt.Sprintf("%s/stations/%d", server.config.APIBasePath, tc.stationID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteStationAPI(t *testing.T) {
	station := randomStation()

	testCases := []struct {
		name          string
		stationID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:      "OK",
			stationID: station.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteStation(mock.AnythingOfType("*gin.Context"), station.ID).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			stationID: station.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteStation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

			url := fmt.Sprintf("%s/stations/%d", server.config.APIBasePath, tc.stationID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomStation() db.ObservationsStation {
	return db.ObservationsStation{
		ID:       util.RandomInt(1, 1000),
		Name:     fmt.Sprintf("%s %s", util.RandomString(12), util.RandomString(8)),
		Lat:      util.NullFloat4{Float4: pgtype.Float4{Float32: util.RandomFloat(-90.0, 90.0), Valid: true}},
		Lon:      util.NullFloat4{Float4: pgtype.Float4{Float32: util.RandomFloat(0.0, 360.0), Valid: true}},
		Province: util.NullString{Text: pgtype.Text{String: util.RandomString(16), Valid: true}},
		Region:   util.NullString{Text: pgtype.Text{String: util.RandomString(16), Valid: true}},
	}
}

func requireBodyMatchStation(t *testing.T, body *bytes.Buffer, station db.ObservationsStation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStation stationResponse
	err = json.Unmarshal(data, &gotStation)
	require.NoError(t, err)
	require.Equal(t, newStationResponse(station), gotStation)
}

func requireBodyMatchStations(t *testing.T, body *bytes.Buffer, stations []db.ObservationsStation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStations listStationsRes
	err = json.Unmarshal(data, &gotStations)
	require.NoError(t, err)

	stationsRes := make([]stationResponse, len(stations))
	for i, stn := range stations {
		stationsRes[i] = newStationResponse(stn)
	}
	require.Equal(t, stationsRes, gotStations.Data)
}
