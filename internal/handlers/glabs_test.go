package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGLabsOptInApi(t *testing.T) {
	glabsOptInRes := randomGlabsOptInRes(t)
	simAccessToken := db.SimAccessToken{
		AccessToken:  glabsOptInRes.AccessToken,
		MobileNumber: fmt.Sprintf("63%s", glabsOptInRes.SubscriberNumber),
		Type:         GLabsAccessTokenType,
	}

	testCases := []struct {
		name          string
		query         gLabsOptInReq
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "SMSOk",
			query: gLabsOptInReq{
				AccessToken:      glabsOptInRes.AccessToken,
				SubscriberNumber: glabsOptInRes.SubscriberNumber,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.FirstOrCreateSimAccessTokenTxParams{
					AccessToken:     simAccessToken.AccessToken,
					AccessTokenType: simAccessToken.Type,
					MobileNumber:    simAccessToken.MobileNumber,
					MobileNumberType: pgtype.Text{
						String: GLabsMobileNumberType,
						Valid:  true,
					},
				}
				store.EXPECT().FirstOrCreateSimAccessTokenTx(mock.AnythingOfType("*gin.Context"), arg).
					Return(db.FirstOrCreateSimAccessTokenTxResult{AccessToken: simAccessToken, IsCreated: true}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchGlabsAccessToken(t, recorder.Body, simAccessToken)
			},
		},
		{
			name: "WebOk",
			query: gLabsOptInReq{
				Code: glabsOptInRes.Code,
			},
			buildStubs: func(store *mockdb.MockStore) {
				mockResponse := map[string]string{
					"access_token":      glabsOptInRes.AccessToken,
					"subscriber_number": glabsOptInRes.SubscriberNumber,
				}
				respBody, _ := json.Marshal(mockResponse)
				httpmock.RegisterResponder(
					http.MethodPost,
					"https://developer.globelabs.com.ph/oauth/access_token",
					httpmock.NewStringResponder(http.StatusOK, string(respBody)),
				)

				arg := db.FirstOrCreateSimAccessTokenTxParams{
					AccessToken:     simAccessToken.AccessToken,
					AccessTokenType: simAccessToken.Type,
					MobileNumber:    simAccessToken.MobileNumber,
					MobileNumberType: pgtype.Text{
						String: GLabsMobileNumberType,
						Valid:  true,
					},
				}
				store.EXPECT().FirstOrCreateSimAccessTokenTx(mock.AnythingOfType("*gin.Context"), arg).
					Return(db.FirstOrCreateSimAccessTokenTxResult{AccessToken: simAccessToken, IsCreated: true}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchGlabsAccessToken(t, recorder.Body, simAccessToken)
			},
		},
		{
			name: "SMSInvalidSubscriberNumber",
			query: gLabsOptInReq{
				AccessToken:      glabsOptInRes.AccessToken,
				SubscriberNumber: "invalid-number",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "FirstOrCreateSimAccessTokenTx", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "SMSInvalidSubscriberNumberLength",
			query: gLabsOptInReq{
				AccessToken:      glabsOptInRes.AccessToken,
				SubscriberNumber: glabsOptInRes.SubscriberNumber[1:8],
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "FirstOrCreateSimAccessTokenTx", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "SMSNoAccessToken",
			query: gLabsOptInReq{
				SubscriberNumber: glabsOptInRes.SubscriberNumber,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "FirstOrCreateSimAccessTokenTx", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "SMSNoSubscriberNumber",
			query: gLabsOptInReq{
				AccessToken: glabsOptInRes.AccessToken,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "FirstOrCreateSimAccessTokenTx", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "MissingParams",
			query: gLabsOptInReq{},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "FirstOrCreateSimAccessTokenTx", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			if tc.query.Code != "" {
				httpmock.Activate()
				defer httpmock.DeactivateAndReset()
			}

			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET("", handler.GLabsOptIn)

			recorder := httptest.NewRecorder()

			url := "/"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to request URL
			q := request.URL.Query()
			q.Add("code", tc.query.Code)
			q.Add("access_token", tc.query.AccessToken)
			q.Add("subscriber_number", tc.query.SubscriberNumber)
			request.URL.RawQuery = q.Encode()

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGLabsUnsubscribeApi(t *testing.T) {
	glabsOptInRes := randomGlabsOptInRes(t)
	simAccessToken := db.SimAccessToken{
		AccessToken:  glabsOptInRes.AccessToken,
		MobileNumber: fmt.Sprintf("63%s", glabsOptInRes.SubscriberNumber),
		Type:         GLabsAccessTokenType,
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "Ok",
			body: gin.H{
				"unsubscribed": gin.H{
					"subscriber_number": simAccessToken.MobileNumber[2:],
					"access_token":      simAccessToken.AccessToken,
					"time_stamp":        time.Now(),
				},
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteSimAccessToken(mock.AnythingOfType("*gin.Context"), simAccessToken.AccessToken).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"unsubscribed": gin.H{
					"subscriber_number": simAccessToken.MobileNumber[2:],
					"access_token":      simAccessToken.AccessToken,
					"time_stamp":        time.Now(),
				},
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteSimAccessToken(mock.AnythingOfType("*gin.Context"), simAccessToken.AccessToken).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "NoAccessToken",
			body: gin.H{
				"unsubscribed": gin.H{
					"subscriber_number": simAccessToken.MobileNumber[2:],
					"time_stamp":        time.Now(),
				},
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "DeleteSimAccessToken", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
			router.POST("", handler.GLabsUnsubscribe)

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

func TestCreateGLabsLoadApi(t *testing.T) {
	gLabsLoad := randomGLabsLoad()

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"outboundRewardRequest": gin.H{
					"transaction_id": gLabsLoad.TransactionID,
					"status":         gLabsLoad.Status,
					"promo":          gLabsLoad.Promo,
					"address":        gLabsLoad.MobileNumber[2:],
				},
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateGLabsLoadParams{
					TransactionID: gLabsLoad.TransactionID,
					Status:        gLabsLoad.Status,
					Promo:         gLabsLoad.Promo,
					MobileNumber:  gLabsLoad.MobileNumber,
				}
				store.EXPECT().GetStationByMobileNumber(mock.AnythingOfType("*gin.Context"),
					pgtype.Text{
						String: arg.MobileNumber,
						Valid:  true,
					},
				).
					Return(db.ObservationsStation{}, nil)
				store.EXPECT().CreateGLabsLoad(mock.AnythingOfType("*gin.Context"), arg).
					Return(gLabsLoad, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchGLabsLoad(t, recorder.Body, gLabsLoad)
			},
		},
		{
			name: "UnknownMobileNumber",
			body: gin.H{
				"outboundRewardRequest": gin.H{
					"transaction_id": gLabsLoad.TransactionID,
					"status":         gLabsLoad.Status,
					"promo":          gLabsLoad.Promo,
					"address":        gLabsLoad.MobileNumber[2:],
				},
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateGLabsLoadParams{
					TransactionID: gLabsLoad.TransactionID,
					Status:        gLabsLoad.Status,
					Promo:         gLabsLoad.Promo,
					MobileNumber:  gLabsLoad.MobileNumber,
				}
				store.EXPECT().GetStationByMobileNumber(mock.AnythingOfType("*gin.Context"),
					pgtype.Text{
						String: arg.MobileNumber,
						Valid:  true,
					},
				).
					Return(db.ObservationsStation{}, db.ErrRecordNotFound)
				store.EXPECT().CreateGLabsLoad(mock.AnythingOfType("*gin.Context"), arg).
					Return(gLabsLoad, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchGLabsLoad(t, recorder.Body, gLabsLoad)
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
			router.POST("", handler.CreateGLabsLoad)

			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
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

func randomGlabsOptInRes(t *testing.T) gLabsOptInReq {
	var g gLabsOptInReq
	err := gofakeit.Struct(&g)
	require.NoError(t, err)
	return g
}

func requireBodyMatchGlabsAccessToken(t *testing.T, body *bytes.Buffer, accessToken db.SimAccessToken) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccesToken gLabsOptInRes
	err = json.Unmarshal(data, &gotAccesToken)

	require.NoError(t, err)
	require.Equal(t, accessToken.AccessToken, gotAccesToken.AccessToken)
	require.Equal(t, accessToken.MobileNumber, gotAccesToken.MobileNumber)
	require.Equal(t, accessToken.Type, gotAccesToken.Type)
}

func randomGLabsLoad() db.GlabsLoad {
	return db.GlabsLoad{
		ID: int64(gofakeit.Number(1, 100)),
		TransactionID: pgtype.Int4{
			Int32: int32(gofakeit.Number(1000000, 9999999)),
			Valid: true,
		},
		Promo: pgtype.Text{
			String: gofakeit.Regex("[A-M]{4,6}[0-9]{2}[05]"),
			Valid:  true,
		},
		Status: pgtype.Text{
			String: gofakeit.RandomString([]string{"SUCCESS", "ERROR"}),
			Valid:  true,
		},
		MobileNumber: gofakeit.Regex("639[0-9]{9}"),
	}
}

func requireBodyMatchGLabsLoad(t *testing.T, body *bytes.Buffer, g db.GlabsLoad) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotGLabsLoad db.GlabsLoad
	err = json.Unmarshal(data, &gotGLabsLoad)

	require.NoError(t, err)
	require.Equal(t, g.TransactionID, gotGLabsLoad.TransactionID)
	require.Equal(t, g.Status, gotGLabsLoad.Status)
	require.Equal(t, g.Promo, gotGLabsLoad.Promo)
	require.Equal(t, g.MobileNumber, gotGLabsLoad.MobileNumber)
}
