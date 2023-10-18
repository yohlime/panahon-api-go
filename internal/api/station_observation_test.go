package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mockdb "github.com/emiliogozo/panahon-api-go/db/mocks"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListStationObservationsAPI(t *testing.T) {
	n := 5
	stationID := util.RandomInt(1, 100)
	stationObsSlice := make([]db.ObservationsObservation, n)
	for i := range stationObsSlice {
		stationObsSlice[i] = randomStationObservation(stationID)
	}

	testCases := []struct {
		name          string
		query         listStationObsReq
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:  "OK",
			query: listStationObsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStationObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(args db.ListStationObservationsParams) bool {
						return !args.IsStartDate && !args.IsEndDate
					})).
					Return(stationObsSlice, nil)
				store.EXPECT().CountStationObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(args db.CountStationObservationsParams) bool {
						return !args.IsStartDate && !args.IsEndDate
					})).
					Return(100, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "WithStartAndEndDate",
			query: listStationObsReq{
				StartDate: "2023-09-01",
				EndDate:   "2023-09-01T23:00:00",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStationObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(args db.ListStationObservationsParams) bool {
						return args.IsStartDate && args.IsEndDate
					})).
					Return(stationObsSlice, nil)
				store.EXPECT().CountStationObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(args db.CountStationObservationsParams) bool {
						return args.IsStartDate && args.IsEndDate
					})).
					Return(50, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "WithInvalidStartAndEndDate",
			query: listStationObsReq{
				StartDate: "notdatetime",
				EndDate:   "invaliddatetime",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStationObservations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "InternalError:ListStationObservations",
			query: listStationObsReq{},
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
			name:  "InternalError:CountStationObservations",
			query: listStationObsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsObservation{}, nil)
				store.EXPECT().CountStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(0, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPage",
			query: listStationObsReq{
				Page: -1,
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
			query: listStationObsReq{
				PerPage: 10000,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStationObservations", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "EmptySlice",
			query: listStationObsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ObservationsObservation{}, nil)
				store.EXPECT().CountStationObservations(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, []db.ObservationsObservation{})
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
			if tc.query.Page != 0 {
				q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			}
			if tc.query.PerPage != 0 {
				q.Add("per_page", fmt.Sprintf("%d", tc.query.PerPage))
			}
			if len(tc.query.StartDate) > 0 {
				q.Add("start_date", tc.query.StartDate)
			}
			if len(tc.query.EndDate) > 0 {
				q.Add("end_date", tc.query.EndDate)
			}
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
				store.EXPECT().GetStationObservation(mock.AnythingOfType("*gin.Context"), arg).
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
				store.EXPECT().GetStationObservation(mock.AnythingOfType("*gin.Context"), arg).
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
				store.EXPECT().GetStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

				store.EXPECT().CreateStationObservation(mock.AnythingOfType("*gin.Context"), arg).
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
				store.EXPECT().CreateStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

				store.EXPECT().UpdateStationObservation(mock.AnythingOfType("*gin.Context"), arg).
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
				store.EXPECT().UpdateStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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
				store.EXPECT().UpdateStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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
				store.EXPECT().DeleteStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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
				store.EXPECT().DeleteStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

func TestListObservationsAPI(t *testing.T) {
	n := 10
	stations := make([]db.ObservationsStation, n)
	stationObsSlice := make([]db.ObservationsObservation, 5)
	var selectedStns []db.ObservationsStation
	var selectedStnIDs []string
	i := 0
	for s := range stations {
		stations[s] = randomStation()
		if (s % 2) == 0 {
			selectedStns = append(selectedStns, stations[s])
			idStr := fmt.Sprintf("%d", stations[s].ID)
			selectedStnIDs = append(selectedStnIDs, idStr)
			if i < 5 {
				stationObsSlice[i] = randomStationObservation[int64](stations[s].ID)
				i++
			}
		}
	}
	nSelected := len(selectedStnIDs)

	testCases := []struct {
		name          string
		query         listObservationsReq
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:  "Default",
			query: listObservationsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), db.ListStationsParams{
					Limit:  pgtype.Int4{Int32: 10, Valid: true},
					Offset: 0,
				}).
					Return(stations, nil)

				store.EXPECT().ListObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(arg db.ListObservationsParams) bool {
						return (arg.Limit.Int32 == 5) && (arg.Offset == 0) && (len(arg.StationIds) == 10)
					}),
				).
					Return(stationObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(100, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "StationIDs",
			query: listObservationsReq{
				StationIDs: strings.Join(selectedStnIDs, ","),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(arg db.ListObservationsParams) bool {
						return (arg.Limit.Int32 == 5) && (arg.Offset == 0) && (len(arg.StationIds) == nSelected)
					}),
				).
					Return(stationObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(50, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "WithStartAndEndDate",
			query: listObservationsReq{
				StartDate: "2023-04-15T12:45:00",
				EndDate:   "2023-04-16",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), db.ListStationsParams{
					Limit:  pgtype.Int4{Int32: 10, Valid: true},
					Offset: 0,
				}).
					Return(stations, nil)

				store.EXPECT().ListObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(arg db.ListObservationsParams) bool {
						return arg.IsStartDate && arg.IsEndDate && (len(arg.StationIds) == 10)
					}),
				).
					Return(stationObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(30, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stationObsSlice)
			},
		},
		{
			name: "WithInvalidStartAndEndDate",
			query: listObservationsReq{
				StartDate: "20230415",
				EndDate:   "invalidDate",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "InternalError:ListStations",
			query: listObservationsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), db.ListStationsParams{
					Limit:  pgtype.Int4{Int32: 10, Valid: true},
					Offset: 0,
				}).
					Return([]db.ObservationsStation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "InternalError:ListObservations",
			query: listObservationsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), db.ListStationsParams{
					Limit:  pgtype.Int4{Int32: 10, Valid: true},
					Offset: 0,
				}).
					Return(stations, nil)

				store.EXPECT().ListObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(arg db.ListObservationsParams) bool {
						return (arg.Limit.Int32 == 5) && (arg.Offset == 0) && (len(arg.StationIds) == 10)
					}),
				).
					Return([]db.ObservationsObservation{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "InternalError:CountObservations",
			query: listObservationsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListStations(mock.AnythingOfType("*gin.Context"), db.ListStationsParams{
					Limit:  pgtype.Int4{Int32: 10, Valid: true},
					Offset: 0,
				}).
					Return(stations, nil)

				store.EXPECT().ListObservations(
					mock.AnythingOfType("*gin.Context"),
					mock.MatchedBy(func(arg db.ListObservationsParams) bool {
						return (arg.Limit.Int32 == 5) && (arg.Offset == 0) && (len(arg.StationIds) == 10)
					}),
				).
					Return(make([]db.ObservationsObservation, 5), nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(0, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPage",
			query: listObservationsReq{
				Page: -1,
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
			query: listObservationsReq{
				PerPage: 10000,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListStations", mock.AnythingOfType("*gin.Context"), mock.Anything)
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

			url := fmt.Sprintf("%s/observations", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to request URL
			q := request.URL.Query()
			if tc.query.Page != 0 {
				q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			}
			if tc.query.PerPage != 0 {
				q.Add("per_page", fmt.Sprintf("%d", tc.query.PerPage))
			}
			if len(tc.query.StationIDs) > 0 {
				q.Add("station_ids", tc.query.StationIDs)
			}
			if len(tc.query.StartDate) > 0 {
				q.Add("start_date", tc.query.StartDate)
			}
			if len(tc.query.EndDate) > 0 {
				q.Add("end_date", tc.query.EndDate)
			}
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomStationObservation[T int | int32 | int64](stationID T) db.ObservationsObservation {
	return db.ObservationsObservation{
		ID:        util.RandomInt[int64](1, 1000),
		StationID: int64(stationID),
		Pres:      pgtype.Float4{Float32: util.RandomFloat[float32](900.0, 1000.0), Valid: true},
		Temp:      pgtype.Float4{Float32: util.RandomFloat[float32](25.0, 35.0), Valid: true},
	}
}

func requireBodyMatchStationObservation(t *testing.T, body *bytes.Buffer, stationObs db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObs StationObservation
	err = json.Unmarshal(data, &gotStationObs)
	require.NoError(t, err)
	require.Equal(t, newStationObservation(stationObs), gotStationObs)
}

func requireBodyMatchStationObservations(t *testing.T, body *bytes.Buffer, stationObsSlice []db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObsSlice paginatedStationObservations
	err = json.Unmarshal(data, &gotStationObsSlice)
	require.NoError(t, err)

	stationObsSliceRes := make([]StationObservation, len(stationObsSlice))
	for i, obs := range stationObsSlice {
		stationObsSliceRes[i] = newStationObservation(obs)
	}
	require.Equal(t, stationObsSliceRes, gotStationObsSlice.Items)
}
