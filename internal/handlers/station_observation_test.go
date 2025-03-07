package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateStationObservationAPI(t *testing.T) {
	stnObs := randomObservation(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"station_id": stnObs.StationID,
				"pres":       stnObs.Pres,
				"temp":       stnObs.Temp,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateStationObservation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CreateStationObservationParams")).
					Return(stnObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stnObs)
			},
		},
		{
			name: "InvalidParam",
			body: gin.H{
				"station_id": stnObs.StationID,
				"pres":       stnObs.Pres,
				"temp":       "32.7",
			},
			buildStubs: func(store *mockdb.MockStore) {},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"station_id": stnObs.StationID,
				"pres":       stnObs.Pres,
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST(":station_id/observations", handler.CreateStationObservation)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/%d/observations", stnObs.StationID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestListStationObservationsAPI(t *testing.T) {
	n := 5
	stationID := gofakeit.Number(1, 100)
	stnObsSlice := make([]db.ObservationsObservation, n)
	stnMOObsSlice := make([]db.ObservationsMoObservation, n)
	for i := range stnObsSlice {
		stnObsSlice[i] = randomObservation(t)
		stnObsSlice[i].StationID = int64(stationID)
		stnMOObsSlice[i] = convertObservationToMOObservation(stnObsSlice[i])
	}

	testCases := []struct {
		name          string
		query         listStationObsReq
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:  "Default",
			query: listStationObsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{}, nil)
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.ListStationObservationsParams")).
					Run(func(ctx context.Context, args db.ListStationObservationsParams) {
						require.False(t, args.IsStartDate)
						require.False(t, args.IsEndDate)
					}).
					Return(stnObsSlice, nil)
				store.EXPECT().CountStationObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountStationObservationsParams")).
					Run(func(ctx context.Context, args db.CountStationObservationsParams) {
						require.False(t, args.IsStartDate)
						require.False(t, args.IsEndDate)
					}).
					Return(100, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
			},
		},
		{
			name:  "MO",
			query: listStationObsReq{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{
						StationType: pgtype.Text{String: "MO", Valid: true},
					}, nil)
				store.EXPECT().ListStationMOObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.ListStationMOObservationsParams")).
					Run(func(ctx context.Context, args db.ListStationMOObservationsParams) {
						require.False(t, args.IsStartDate)
						require.False(t, args.IsEndDate)
					}).
					Return(stnMOObsSlice, nil)
				store.EXPECT().CountStationMOObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountStationMOObservationsParams")).
					Run(func(ctx context.Context, args db.CountStationMOObservationsParams) {
						require.False(t, args.IsStartDate)
						require.False(t, args.IsEndDate)
					}).
					Return(100, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
			},
		},
		{
			name: "WithStartAndEndDate",
			query: listStationObsReq{
				StartDate: "2023-09-01",
				EndDate:   "2023-09-01T23:00:00",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{}, nil)
				store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.ListStationObservationsParams")).
					Run(func(ctx context.Context, args db.ListStationObservationsParams) {
						require.True(t, args.IsStartDate)
						require.True(t, args.IsEndDate)
					}).
					Return(stnObsSlice, nil)
				store.EXPECT().CountStationObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountStationObservationsParams")).
					Run(func(ctx context.Context, args db.CountStationObservationsParams) {
						require.True(t, args.IsStartDate)
						require.True(t, args.IsEndDate)
					}).
					Return(50, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
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
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{}, nil)
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
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{}, nil)
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
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.ObservationsStation{}, nil)
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET(":station_id/observations", handler.ListStationObservations)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d/observations", stationID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

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

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetStationObservationAPI(t *testing.T) {
	stnObs := randomObservation(t)

	testCases := []struct {
		name          string
		params        db.GetStationObservationParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			params: db.GetStationObservationParams{
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStationObservation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.GetStationObservationParams")).
					Return(stnObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stnObs)
			},
		},
		{
			name: "NotFound",
			params: db.GetStationObservationParams{
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStationObservation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.GetStationObservationParams")).
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
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
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
				StationID: stnObs.StationID,
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET(":station_id/observations/:id", handler.GetStationObservation)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d/observations/%d", tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestUpdateStationObservationAPI(t *testing.T) {
	stnObs := randomObservation(t)

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
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
			},
			body: gin.H{
				"id":         stnObs.ID,
				"station_id": stnObs.StationID,
				"pres":       stnObs.Pres,
				"temp":       stnObs.Temp,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateStationObservation(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.UpdateStationObservationParams")).
					Return(stnObs, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservation(t, recorder.Body, stnObs)
			},
		},
		{
			name: "InternalError",
			params: db.UpdateStationObservationParams{
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
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
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.PUT(":station_id/observations/:id", handler.UpdateStationObservation)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/%d/observations/%d", tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteStationObservationObservationAPI(t *testing.T) {
	stnObs := randomObservation(t)

	testCases := []struct {
		name          string
		params        db.DeleteStationObservationParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			params: db.DeleteStationObservationParams{
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
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
			params: db.DeleteStationObservationParams{
				ID:        stnObs.ID,
				StationID: stnObs.StationID,
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.DELETE(":station_id/observations/:id", handler.DeleteStationObservation)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d/observations/%d", tc.params.StationID, tc.params.ID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestListObservationsAPI(t *testing.T) {
	n := 10
	stations := make([]db.ObservationsStation, n)
	stnObsSlice := make([]db.ObservationsObservation, 5)
	var selectedStns []db.ObservationsStation
	var selectedStnIDs []string
	i := 0
	for s := range stations {
		stations[s] = randomStation(t)
		if (s % 2) == 0 {
			selectedStns = append(selectedStns, stations[s])
			idStr := fmt.Sprintf("%d", stations[s].ID)
			selectedStnIDs = append(selectedStnIDs, idStr)
			if i < 5 {
				stnObsSlice[i] = randomObservation(t)
				stnObsSlice[i].StationID = stations[s].ID
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
					Return(stnObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(100, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
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
					Return(stnObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(50, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
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
					Return(stnObsSlice, nil)

				store.EXPECT().CountObservations(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CountObservationsParams")).
					Return(30, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchStationObservations(t, recorder.Body, stnObsSlice)
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET("", handler.ListObservations)

			recorder := httptest.NewRecorder()

			url := "/"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

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

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetNearestLatestStationObservationAPI(t *testing.T) {
	testCases := []struct {
		name          string
		query         getNearestLatestStationObsReq
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			query: getNearestLatestStationObsReq{
				Pt: "12.5,121.6",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetNearestLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetNearestLatestStationObservationRow{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "NotFound",
			query: getNearestLatestStationObsReq{
				Pt: "12.5,121.6",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetNearestLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetNearestLatestStationObservationRow{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			query: getNearestLatestStationObsReq{
				Pt: "12.5,121.6",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetNearestLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetNearestLatestStationObservationRow{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPt",
			query: getNearestLatestStationObsReq{
				Pt: "12.5",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetNearestLatestStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET("", handler.GetNearestLatestStationObservation)

			recorder := httptest.NewRecorder()

			url := "/"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			q := request.URL.Query()
			q.Add("pt", tc.query.Pt)
			request.URL.RawQuery = q.Encode()

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetLatestStationObservationAPI(t *testing.T) {
	stnObs := randomObservation(t)

	testCases := []struct {
		name          string
		stationID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:      "OK",
			stationID: stnObs.StationID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetLatestStationObservationRow{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			stationID: stnObs.StationID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetLatestStationObservationRow{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			stationID: stnObs.StationID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetLatestStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.GetLatestStationObservationRow{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InvalidID",
			stationID: -3,
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetLatestStationObservation", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET(":station_id", handler.GetLatestStationObservation)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d", tc.stationID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomObservation(t *testing.T) db.ObservationsObservation {
	var o models.StationObservation
	err := gofakeit.Struct(&o)
	require.NoError(t, err)

	return db.ObservationsObservation{
		ID:        o.ID,
		StationID: o.StationID,
		Pres:      util.ToFloat4(o.Pres),
		Temp:      util.ToFloat4(o.Temp),
	}
}

func requireBodyMatchStationObservation(t *testing.T, body *bytes.Buffer, stationObs db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObs models.StationObservation
	err = json.Unmarshal(data, &gotStationObs)
	require.NoError(t, err)
	require.Equal(t, models.NewStationObservation(stationObs), gotStationObs)
}

func requireBodyMatchStationObservations(t *testing.T, body *bytes.Buffer, stationObsSlice []db.ObservationsObservation) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotStationObsSlice paginatedStationObservations
	err = json.Unmarshal(data, &gotStationObsSlice)
	require.NoError(t, err)

	stationObsSliceRes := make([]models.StationObservation, len(stationObsSlice))
	for i, obs := range stationObsSlice {
		stationObsSliceRes[i] = models.NewStationObservation(obs)
	}
	require.Equal(t, stationObsSliceRes, gotStationObsSlice.Items)
}
