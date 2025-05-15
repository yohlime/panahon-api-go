package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCSIStoreMisol(t *testing.T) {
	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"weather": "75112112108101,123.8854,10.3157,1726809924,31,91,1007,6.7,13.4,105,1718,77,30.0001,2,23,3.7,3.8,0.012,17.8,0.065,50,10,25.2,54.4,0,90,1",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetMisolStation(mock.AnythingOfType("*gin.Context"), int64(75112112108101)).
					Return(db.MisolStation{ID: 17, StationID: 139}, nil)
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), int64(139)).
					Return(db.ObservationsStation{}, nil)
				store.EXPECT().CreateStationObservation(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsObservation{}, nil)
				store.EXPECT().CreateStationHealth(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsStationhealth{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "MisolStationNotFound",
			body: gin.H{
				"weather": "75112112108101,123.8854,10.3157,1726809924,31,91,1007,6.7,13.4,105,1718,77,30.0001,2,23,3.7,3.8,0.012,17.8,0.065,50,10,25.2,54.4,0,90,1",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetMisolStation(mock.AnythingOfType("*gin.Context"), int64(75112112108101)).
					Return(db.MisolStation{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "StationNotFound",
			body: gin.H{
				"weather": "75112112108101,123.8854,10.3157,1726809924,31,91,1007,6.7,13.4,105,1718,77,30.0001,2,23,3.7,3.8,0.012,17.8,0.065,50,10,25.2,54.4,0,90,1",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetMisolStation(mock.AnythingOfType("*gin.Context"), int64(75112112108101)).
					Return(db.MisolStation{ID: 17, StationID: 139}, nil)
				store.EXPECT().GetStation(mock.AnythingOfType("*gin.Context"), int64(139)).
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST("", handler.CSIStoreMisol)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}
