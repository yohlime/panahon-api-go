package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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
		UserID:    user.ID,
		Name:      "Test Token",
		TokenHash: apiTok.GetHash(),
	}

	authUser := token.User{
		Username: user.Username,
	}

	// The owner's entitlement: 2 stations, 2 variables, 730 days.
	entPerms := models.APITokenPermissions{
		StationIDs:     []int64{11, 22},
		Variables:      []string{"rh", "temp"},
		MaxDaysHistory: 730,
	}
	entStations, entVariables := models.MarshalUserAccess(models.UserAccess{
		UserID:         user.ID,
		StationIDs:     entPerms.StationIDs,
		Variables:      entPerms.Variables,
		MaxDaysHistory: entPerms.MaxDaysHistory,
	})
	dbAccess := db.UserAccess{
		UserID:         user.ID,
		StationIds:     entStations,
		Variables:      entVariables,
		MaxDaysHistory: 730,
	}

	// Requested scope exceeds the entitlement, so it is clamped to it:
	// stations [11,33] ∩ [11,22] = [11]; variables [rain,temp] ∩ [rh,temp] = [temp].
	clampedPerms := models.APITokenPermissions{
		StationIDs:     []int64{11},
		Variables:      []string{"temp"},
		MaxDaysHistory: 365,
	}

	mustJSON := func(v any) []byte {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return b
	}

	// Entitlements with no stations vs. all stations (nil means all).
	dbAccessEmpty := db.UserAccess{
		UserID:         user.ID,
		StationIds:     []byte("[]"),
		Variables:      entVariables,
		MaxDaysHistory: 730,
	}
	dbAccessAll := db.UserAccess{
		UserID:         user.ID,
		MaxDaysHistory: 730,
	}
	allPerms := models.APITokenPermissions{MaxDaysHistory: 730}
	dbAPITokAll := dbAPITok
	dbAPITokAll.Permissions = mustJSON(allPerms)
	apiTokenAll := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITokAll),
		Token:    apiTok.GetValue(),
	}

	dbAPITokEntitled := dbAPITok
	dbAPITokEntitled.Permissions = mustJSON(entPerms)
	apiTokenEntitled := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITokEntitled),
		Token:    apiTok.GetValue(),
	}

	dbAPITokClamped := dbAPITok
	dbAPITokClamped.Permissions = mustJSON(clampedPerms)
	apiTokenClamped := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITokClamped),
		Token:    apiTok.GetValue(),
	}

	// Admin-owned tokens: default to everything (all stations/variables, no
	// history limit) and never clamp an explicit scope.
	adminAllPerms := models.APITokenPermissions{MaxDaysHistory: -1}
	dbAPITokAdminAll := dbAPITok
	dbAPITokAdminAll.Permissions = mustJSON(adminAllPerms)
	apiTokenAdminAll := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITokAdminAll),
		Token:    apiTok.GetValue(),
	}

	adminScopedPerms := models.APITokenPermissions{
		StationIDs:     []int64{5},
		Variables:      []string{"rain"},
		MaxDaysHistory: 1825,
	}
	dbAPITokAdminScoped := dbAPITok
	dbAPITokAdminScoped.Permissions = mustJSON(adminScopedPerms)
	apiTokenAdminScoped := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbAPITokAdminScoped),
		Token:    apiTok.GetValue(),
	}

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "DefaultUsesEntitlement",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccess, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					if err := json.Unmarshal(arg.Permissions, &permissions); err != nil {
						return false
					}
					return arg.UserID == user.ID && !arg.ExpiresAt.Valid && reflect.DeepEqual(permissions, entPerms)
				})).Return(dbAPITokEntitled, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiTokenEntitled)
			},
		},
		{
			name: "WithPermissionsClamped",
			body: gin.H{
				"name": dbAPITok.Name,
				"permissions": gin.H{
					"variables":        []string{"rain", "temp"},
					"station_ids":      []int64{11, 33},
					"max_days_history": 365,
				},
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccess, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					if err := json.Unmarshal(arg.Permissions, &permissions); err != nil {
						return false
					}
					return !arg.ExpiresAt.Valid && reflect.DeepEqual(permissions, clampedPerms)
				})).Return(dbAPITokClamped, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiTokenClamped)
			},
		},
		{
			name: "AdminDefaultsToAll",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"ADMIN"}, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					if err := json.Unmarshal(arg.Permissions, &permissions); err != nil {
						return false
					}
					return reflect.DeepEqual(permissions, adminAllPerms)
				})).Return(dbAPITokAdminAll, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiTokenAdminAll)
			},
		},
		{
			name: "AdminExplicitScopeIsNotClamped",
			body: gin.H{
				"name": dbAPITok.Name,
				"permissions": gin.H{
					"variables":        []string{"rain"},
					"station_ids":      []int64{5},
					"max_days_history": 1825,
				},
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"SUPERADMIN"}, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					if err := json.Unmarshal(arg.Permissions, &permissions); err != nil {
						return false
					}
					return reflect.DeepEqual(permissions, adminScopedPerms)
				})).Return(dbAPITokAdminScoped, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiTokenAdminScoped)
			},
		},
		{
			name: "NoStationAccess",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(db.UserAccess{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "EmptyStationGrant",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccessEmpty, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "NoStationsRequested",
			body: gin.H{
				"name": dbAPITok.Name,
				"permissions": gin.H{
					"station_ids": []int64{},
					"variables":   []string{"temp"},
				},
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccess, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "AllStationsGrant",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccessAll, nil)
				store.EXPECT().CreateAPIToken(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateAPITokenParams) bool {
					var permissions models.APITokenPermissions
					if err := json.Unmarshal(arg.Permissions, &permissions); err != nil {
						return false
					}
					return reflect.DeepEqual(permissions, allPerms)
				})).Return(dbAPITokAll, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchAPIToken(t, recorder.Body, apiTokenAll)
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
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "StoreTokenError",
			body: gin.H{
				"name": dbAPITok.Name,
			},
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().GetUserAccess(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(dbAccess, nil)
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
			// The entitlement is cached process-wide; drop it so each case loads it.
			userAccessCache.del(user.ID)

			store := mockdb.NewMockStore(t)
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(tokenMaker, store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST("/:id/api-tokens",
				middlewares.AuthMiddleware(tokenMaker, store, false),
				handler.CreateAPIToken)

			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/%d/api-tokens", user.ID)
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
	for t := range dbAPITokens {
		apiTok := util.NewAPIToken("")
		dbAPITokens[t] = db.APIToken{
			UserID:    user.ID,
			Name:      util.RandomString(12),
			TokenHash: apiTok.GetHash(),
		}
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
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(tokenMaker, store)

			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.GET("/:id/api-tokens",
				middlewares.AuthMiddleware(tokenMaker, store, false),
				handler.ListAPITokens)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d/api-tokens", user.ID)
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
			name: "InvalidTokenID",
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
			name:    "TokenNotFound",
			tokenID: dbAPITok.ID,
			setupAuth: func(request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker, store *mockdb.MockStore) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
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
			router.DELETE("/:id/api-tokens/:token_id",
				middlewares.AuthMiddleware(tokenMaker, store, false),
				handler.DeleteAPIToken)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d/api-tokens/%d", user.ID, tc.tokenID)
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
