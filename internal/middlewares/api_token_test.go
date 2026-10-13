package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAPITokenMiddleware(t *testing.T) {
	testCases := []struct {
		name          string
		permissive    bool
		setupAuth     func(request *http.Request)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(request *http.Request) {
				tok := util.NewAPIToken("")
				addAuthorization(request, AuthTypeQuery, tok.GetValue())
			},
			buildStubs: func(store *mockdb.MockStore) {
				dbTok := db.APIToken{
					ID: util.RandomInt[int64](1, 1000),
					ExpiresAt: pgtype.Timestamptz{
						Time: time.Now().Add(60 * time.Minute),
					},
				}
				store.EXPECT().GetAPITokenByHash(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("string")).
					Return(dbTok, nil)
				store.EXPECT().UpdateAPITokenLastUsedAt(mock.AnythingOfType("context.backgroundCtx"), dbTok.ID).Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:       "NoAuthorization",
			setupAuth:  func(request *http.Request) {},
			buildStubs: func(store *mockdb.MockStore) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(request *http.Request) {
				addAuthorization(request, "unsupported", "")
			},
			buildStubs: func(store *mockdb.MockStore) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)

			tc.buildStubs(store)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				APITokenMiddleware(store, tc.permissive),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(request)
			router.ServeHTTP(recorder, request)
			time.Sleep(100 * time.Millisecond)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestStationMiddleware(t *testing.T) {
	stnID := util.RandomInt[int64](1, 1000)
	testCases := []struct {
		name          string
		setupAccess   func(ctx *gin.Context)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAccess: func(ctx *gin.Context) {
				perm := models.APITokenPermissions{
					StationIDs: []int64{int64(stnID)},
				}
				data, _ := json.Marshal(perm)

				tok := db.APIToken{
					Permissions: data,
				}
				ctx.Set(models.APITokenAuthKey, &tok)
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "AdminBypass",
			setupAccess: func(ctx *gin.Context) {
				ctx.Set(models.AuthPayloadKey, &token.Payload{
					User: token.User{Roles: []string{string(models.AdminRole)}},
				})
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "TokenMissing",
			setupAccess: func(ctx *gin.Context) {
				ctx.Set(models.APITokenAuthKey, nil)
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "NoAccess",
			setupAccess: func(ctx *gin.Context) {
				perm := models.APITokenPermissions{
					StationIDs: []int64{int64(stnID + 10)},
				}
				data, _ := json.Marshal(perm)

				tok := db.APIToken{
					Permissions: data,
				}
				ctx.Set(models.APITokenAuthKey, &tok)
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			router := gin.Default()
			router.GET(
				"/stations/:station_id",
				tc.setupAccess,
				StationMiddleware(""),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/stations/%d", stnID), nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			time.Sleep(100 * time.Millisecond)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestRequireAdminOrAPIToken(t *testing.T) {
	testCases := []struct {
		name          string
		setupAccess   func(ctx *gin.Context)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "AdminSession",
			setupAccess: func(ctx *gin.Context) {
				ctx.Set(models.AuthPayloadKey, &token.Payload{
					User: token.User{Roles: []string{string(models.SuperAdminRole)}},
				})
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "ValidAPIToken",
			setupAccess: func(ctx *gin.Context) {
				ctx.Set(models.APITokenAuthKey, &db.APIToken{ID: 1})
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "NoAuth",
			setupAccess: func(ctx *gin.Context) {
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "NonAdminSession",
			setupAccess: func(ctx *gin.Context) {
				ctx.Set(models.AuthPayloadKey, &token.Payload{
					User: token.User{Roles: []string{"USER"}},
				})
				ctx.Next()
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			router := gin.Default()
			router.GET(
				"/observations",
				tc.setupAccess,
				RequireAdminOrAPIToken(),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/observations", nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
