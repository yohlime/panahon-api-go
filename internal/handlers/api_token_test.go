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

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/middlewares"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateAPIToken(t *testing.T) {
	user, _, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)

	apiTok := util.NewAPIToken("")
	dbAPITok := db.APIToken{
		Name:      "Test Token",
		TokenHash: apiTok.GetHash(),
	}
	apiToken := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITok),
		Token:    apiTok.GetValue(),
	}

	authUser := token.User{
		Username: user.Username,
	}

	nStn := 5
	stationIDs := make([]int64, nStn)
	for s := range stationIDs {
		stationIDs[s] = gofakeit.Int64()
	}
	varNames := []string{"rain", "temp"}

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "Default",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					err := json.Unmarshal(arg.Permissions, &permissions)
					if err != nil {
						return false
					}
					return !arg.ExpiresAt.Valid && permissions.MaxDaysHistory == models.DefaultMaxDaysHistory && len(permissions.StationIDs) == 0 && len(permissions.Variables) == len(models.DefaultAPITokenVariables)
				})).Return(dbAPITok, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiToken)
			},
		},
		{
			name: "WithPermissions",
			body: gin.H{
				"name": dbAPITok.Name,
				"permissions": gin.H{
					"variables":   varNames,
					"station_ids": stationIDs,
				},
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				newDBAPITok := dbAPITok
				perm, err := json.Marshal(models.APITokenPermissions{
					StationIDs:     stationIDs,
					Variables:      varNames,
					MaxDaysHistory: 365,
				})
				require.NoError(t, err)
				newDBAPITok.Permissions = perm
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					err := json.Unmarshal(arg.Permissions, &permissions)
					if err != nil {
						return false
					}
					return !arg.ExpiresAt.Valid && permissions.MaxDaysHistory == models.DefaultMaxDaysHistory && len(permissions.StationIDs) == nStn && len(permissions.Variables) == len(varNames)
				})).Return(newDBAPITok, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				newAPIToken := apiToken
				newAPIToken.Permissions = models.APITokenPermissions{
					StationIDs:     stationIDs,
					Variables:      varNames,
					MaxDaysHistory: 365,
				}
				requireBodyMatchAPIToken(t, recorder.Body, newAPIToken)
			},
		},
		{
			name: "EmptyBody",
			body: gin.H{},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Unauthenticated",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth:  func(request *http.Request) {},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "StoreTokenError",
			body: gin.H{
				"name": dbAPITok.Name,
				"permissions": gin.H{
					"variables":   varNames,
					"station_ids": stationIDs,
				},
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("db.CreateAPITokenParams")).
					Return(db.APIToken{}, sql.ErrConnDone)
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
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(tokenMaker, store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST("",
				middlewares.AuthMiddleware(tokenMaker, false),
				handler.CreateAPIToken)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(request)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func TestListAPITokens(t *testing.T) {
	user, _, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)

	n := 3
	dbAPITokens := make([]db.APIToken, n)
	apiTokens := make([]models.APITokenWithSecret, n)
	for t := range dbAPITokens {
		apiTok := util.NewAPIToken("")
		dbAPITok := db.APIToken{
			Name:      util.RandomString(12),
			TokenHash: apiTok.GetHash(),
		}
		dbAPITokens[t] = dbAPITok
		apiToken := models.APITokenWithSecret{
			APIToken: models.NewAPIToken(dbAPITok),
			Token:    apiTok.GetValue(),
		}
		apiTokens[t] = apiToken
	}

	authUser := token.User{
		Username: user.Username,
	}

	testCases := []struct {
		name          string
		setupAuth     func(request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "Default",
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().ListAPITokensByUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(dbAPITokens, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:       "Unauthenticated",
			setupAuth:  func(request *http.Request) {},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UserNotFound",
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(db.User{}, db.ErrRecordNotFound)
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
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(tokenMaker, store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET("",
				middlewares.AuthMiddleware(tokenMaker, false),
				handler.ListAPITokens)

			recorder := httptest.NewRecorder()

			url := "/"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(request)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteAPIToken(t *testing.T) {
	user, _, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)

	apiTok := util.NewAPIToken("")
	dbAPITok := db.APIToken{
		ID:        util.RandomInt[int64](111, 333),
		UserID:    user.ID,
		Name:      "Test Token",
		TokenHash: apiTok.GetHash(),
	}

	authUser := token.User{
		Username: user.Username,
	}

	nStn := 5
	stationIDs := make([]int64, nStn)
	for s := range stationIDs {
		stationIDs[s] = gofakeit.Int64()
	}

	testCases := []struct {
		name          string
		tokenID       int64
		setupAuth     func(request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:    "Default",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().GetAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).
					Return(dbAPITok, nil)
				store.EXPECT().DeleteAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name: "NoTokenID",
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "Unauthenticated",
			tokenID:    dbAPITok.ID,
			setupAuth:  func(request *http.Request) {},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:    "UserNotFound",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:    "TokenNotFound",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().GetAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).
					Return(db.APIToken{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:    "WrongOwner",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				wrongDBAPITok := dbAPITok
				wrongDBAPITok.UserID = dbAPITok.UserID + 1
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().GetAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).
					Return(wrongDBAPITok, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:    "DeleteTokenError",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().GetAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).
					Return(dbAPITok, nil)
				store.EXPECT().DeleteAPIToken(mock.AnythingOfType("*gin.Context"), dbAPITok.ID).Return(fmt.Errorf("error"))
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
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(tokenMaker, store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.DELETE(":token_id",
				middlewares.AuthMiddleware(tokenMaker, false),
				handler.DeleteAPIToken)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d", tc.tokenID)
			if tc.tokenID == 0 {
				url = "/"
			}
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			tc.setupAuth(request)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func requireBodyMatchAPIToken(t *testing.T, body *bytes.Buffer, apiToken models.APITokenWithSecret) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAPIToken models.APITokenWithSecret
	err = json.Unmarshal(data, &gotAPIToken)

	require.NoError(t, err)
	require.Equal(t, apiToken.Name, gotAPIToken.Name)
	require.Equal(t, apiToken.Permissions, gotAPIToken.Permissions)
}
