package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func cursorObservations(stationID int64) []db.ObservationsObservation {
	now := time.Now().UTC().Truncate(time.Second)
	return []db.ObservationsObservation{
		{ID: 30, StationID: stationID, Timestamp: pgtype.Timestamptz{Time: now, Valid: true}},
		{ID: 20, StationID: stationID, Timestamp: pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true}},
		{ID: 10, StationID: stationID, Timestamp: pgtype.Timestamptz{Time: now.Add(-2 * time.Minute), Valid: true}},
	}
}

func TestListStationObservationsCursorAPI(t *testing.T) {
	stationID := int64(7)
	obs := cursorObservations(stationID)
	cursor := encodeObservationCursor(obs[0].Timestamp.Time.Add(time.Minute), 40)

	store := mockdb.NewMockStore(t)
	store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), stationID).
		Return(db.ObservationsStation{}, nil)
	store.EXPECT().ListStationObservations(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.ListStationObservationsParams) bool {
		return arg.Limit.Valid && arg.Limit.Int32 == 3 &&
			arg.Offset == 0 &&
			arg.CursorTs.Valid && arg.CursorID.Valid && arg.CursorID.Int64 == 40
	})).Return(obs, nil)

	handler := newTestHandler(store, nil)
	router := gin.Default()
	router.GET(":station_id/observations", adminContext(), handler.ListStationObservations)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%d/observations?limit=2&cursor=%s", stationID, url.QueryEscape(cursor)), nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var got util.CursorList[models.StationObservation]
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.True(t, got.HasMore)
	require.Len(t, got.Items, 2)
	require.Equal(t, int64(30), got.Items[0].ID)
	require.Equal(t, int64(20), got.Items[1].ID)

	nextTime, nextID, err := decodeObservationCursor(got.NextCursor)
	require.NoError(t, err)
	require.Equal(t, int64(20), nextID)
	require.True(t, nextTime.Equal(obs[1].Timestamp.Time))
	store.AssertNotCalled(t, "CountStationObservations", mock.Anything, mock.Anything)
}

func TestListObservationsCursorAPI(t *testing.T) {
	obs := cursorObservations(7)
	cursor := encodeObservationCursor(obs[0].Timestamp.Time.Add(time.Minute), 40)

	store := mockdb.NewMockStore(t)
	store.EXPECT().ListObservations(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.ListObservationsParams) bool {
		return arg.Limit.Valid && arg.Limit.Int32 == 3 &&
			arg.Offset == 0 &&
			arg.CursorTs.Valid && arg.CursorID.Valid && arg.CursorID.Int64 == 40
	})).Return(obs, nil)

	handler := newTestHandler(store, nil)
	router := gin.Default()
	router.GET("/observations", adminContext(), handler.ListObservations)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/observations?limit=2&station_ids=7&cursor=%s", url.QueryEscape(cursor)),
		nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var got util.CursorList[models.StationObservation]
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.True(t, got.HasMore)
	require.Len(t, got.Items, 2)
	require.Equal(t, int64(30), got.Items[0].ID)
	require.Equal(t, int64(20), got.Items[1].ID)
	store.AssertNotCalled(t, "CountObservations", mock.Anything, mock.Anything)
}

func TestListStationObservationsInvalidCursor(t *testing.T) {
	store := mockdb.NewMockStore(t)
	handler := newTestHandler(store, nil)
	router := gin.Default()
	router.GET(":station_id/observations", adminContext(), handler.ListStationObservations)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/7/observations?cursor=not-a-cursor", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	store.AssertNotCalled(t, "GetStation", mock.Anything, mock.Anything)
	store.AssertNotCalled(t, "ListStationObservations", mock.Anything, mock.Anything)
}
