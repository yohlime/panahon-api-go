package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/sensor"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPromoTexterStoreLufft(t *testing.T) {
	mobileNum := util.RandomMobileNumber()
	var lufft sensor.Lufft
	gofakeit.Struct(&lufft)
	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"number": mobileNum,
				"msg":    lufft.String(23),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStationByMobileNumber(mock.AnythingOfType("*gin.Context"), mock.Anything).
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
			name: "NotFound",
			body: gin.H{
				"number": mobileNum,
				"msg":    lufft.String(23),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStationByMobileNumber(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.ObservationsStation{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"number": mobileNum,
				"msg":    lufft.String(23),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetStationByMobileNumber(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST("", handler.PromoTexterStoreLufft)

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
