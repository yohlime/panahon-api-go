package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	AuthTypeBearer = "bearer"
	AuthTypeCookie = "cookie"
	AuthTypeQuery  = "query"
)

func TestAuthMiddleware(t *testing.T) {
	tokenStr := util.RandomString(24)
	payload := token.Payload{}
	testCases := []struct {
		name          string
		permissive    bool
		setupAuth     func(t *testing.T, request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "AuthorizationBearer",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "AuthorizationCookie",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeCookie, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:       "NoAuthorization",
			setupAuth:  func(t *testing.T, request *http.Request) {},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, "unsupported", "")
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthBearerFormat",
			setupAuth: func(t *testing.T, request *http.Request) {
				authorizationHeader := fmt.Sprintf("%s%s", AuthTypeBearer, util.RandomString(12))
				request.Header.Set(models.AuthHeaderKey, authorizationHeader)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredBearerToken",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(nil, token.ErrExpiredToken)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tokenMaker := mocktoken.NewMockMaker(t)
			store := mockdb.NewMockStore(t)

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, store, tc.permissive),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request)
			router.ServeHTTP(recorder, request)
			time.Sleep(100 * time.Millisecond)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestRoleMiddleware(t *testing.T) {
	tokenStr := util.RandomString(32)
	testCases := []struct {
		name          string
		role          string
		setupAuth     func(t *testing.T, request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			role: "USER",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{"USER", "VIEWER"}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "MissingRole",
			role: string(models.AdminRole),
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tokenMaker := mocktoken.NewMockMaker(t)
			store := mockdb.NewMockStore(t)

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, store, false),
				RoleMiddleware(tc.role),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestAdminMiddleware(t *testing.T) {
	tokenStr := util.RandomString(32)
	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request)
		buildStubs    func(tokenMaker *mocktoken.MockMaker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "IsAdmin",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{string(models.AdminRole)}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "IsSuperAdmin",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{string(models.SuperAdminRole)}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "NotAdmin",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{"USER", "VIEWER"}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "NoRoles",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {
				payload := token.Payload{User: token.User{Roles: []string{}}}
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&payload, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tokenMaker := mocktoken.NewMockMaker(t)
			store := mockdb.NewMockStore(t)

			router := gin.Default()

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, store, false),
				AdminMiddleware(),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestValidateAPIToken(t *testing.T) {
	testCases := []struct {
		name          string
		apiToken      util.APIToken
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, token *db.APIToken, err error)
	}{
		{
			name:     "OK",
			apiToken: *util.NewAPIToken(""),
			buildStubs: func(store *mockdb.MockStore) {
				dbTok := db.APIToken{
					ID: util.RandomInt[int64](1, 1000),
					ExpiresAt: pgtype.Timestamptz{
						Time: time.Now().Add(60 * time.Minute),
					},
				}
				store.EXPECT().GetAPITokenByHash(mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
					Return(dbTok, nil)
				store.EXPECT().UpdateAPITokenLastUsedAt(mock.AnythingOfType("context.backgroundCtx"), dbTok.ID).Return(nil)
			},
			checkResponse: func(t *testing.T, token *db.APIToken, err error) {
				require.NoError(t, err)
				require.NotNil(t, token)
			},
		},
		{
			name:     "TokenNotFound",
			apiToken: *util.NewAPIToken(util.RandomString(24)),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAPITokenByHash(mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
					Return(db.APIToken{}, db.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, token *db.APIToken, err error) {
				require.Error(t, err)
				require.Nil(t, token)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)

			tc.buildStubs(store)

			tok, err := validateAPIToken(context.Background(), store, tc.apiToken)

			time.Sleep(100 * time.Millisecond)
			tc.checkResponse(t, tok, err)
		})
	}
}

// TestRequireSelfOrAdmin verifies the admin-or-self guard: admins pass, a user
// passes only for their own id, and everyone else is rejected.
func TestRequireSelfOrAdmin(t *testing.T) {
	caller := db.User{ID: 7, Username: "caller"}

	testCases := []struct {
		name        string
		callerRoles []string
		hasAuth     bool
		targetID    string
		buildStubs  func(store *mockdb.MockStore)
		wantStatus  int
	}{
		{
			name:        "AdminAllowed",
			callerRoles: []string{string(models.AdminRole)},
			hasAuth:     true,
			targetID:    "99",
			buildStubs:  func(store *mockdb.MockStore) {},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "SuperAdminAllowed",
			callerRoles: []string{string(models.SuperAdminRole)},
			hasAuth:     true,
			targetID:    "99",
			buildStubs:  func(store *mockdb.MockStore) {},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "SelfAllowed",
			callerRoles: []string{"USER"},
			hasAuth:     true,
			targetID:    "7",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(caller, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "OtherDenied",
			callerRoles: []string{"USER"},
			hasAuth:     true,
			targetID:    "99",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(caller, nil)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Unauthenticated",
			hasAuth:    false,
			targetID:   "7",
			buildStubs: func(store *mockdb.MockStore) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "InvalidTargetID",
			callerRoles: []string{"USER"},
			hasAuth:     true,
			targetID:    "abc",
			buildStubs:  func(store *mockdb.MockStore) {},
			wantStatus:  http.StatusBadRequest,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			router := gin.Default()
			router.GET(
				"/users/:id/things",
				func(ctx *gin.Context) {
					if tc.hasAuth {
						ctx.Set(models.AuthPayloadKey, &token.Payload{
							User: token.User{Username: "caller", Roles: tc.callerRoles},
						})
					}
					ctx.Next()
				},
				RequireSelfOrAdmin(store, "id"),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, "/users/"+tc.targetID+"/things", nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}

func addAuthorization(
	request *http.Request,
	authType string,
	token string,
) {
	if authType == AuthTypeBearer {
		authorizationHeader := fmt.Sprintf("%s %s", authType, token)
		request.Header.Set(models.AuthHeaderKey, authorizationHeader)
	} else if authType == AuthTypeCookie {
		addAccessTokenCookie(request, token)
	} else if authType == AuthTypeQuery {
		request.URL.RawQuery = fmt.Sprintf("%s=%s", models.APIKeyQueryParam, token)
	}
}

func addAccessTokenCookie(request *http.Request, token string) {
	request.AddCookie(&http.Cookie{Name: models.AccessTokenCookieName, Value: token})
}

// TestAuthMiddlewareSessionRevocation verifies that a token bound to a session
// stops authenticating once the session is gone (logged out), even though the
// token itself is still unexpired.
func TestAuthMiddlewareSessionRevocation(t *testing.T) {
	testCases := []struct {
		name       string
		session    db.Session
		sessionErr error
		wantStatus int
	}{
		{
			name:       "LiveSession",
			session:    db.Session{IsBlocked: false, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "DeletedSession",
			sessionErr: db.ErrRecordNotFound,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "BlockedSession",
			session:    db.Session{IsBlocked: true, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			sessionID := uuid.New()
			payload := token.Payload{SessionID: sessionID}

			tokenMaker := mocktoken.NewMockMaker(t)
			store := mockdb.NewMockStore(t)

			tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&payload, nil)
			store.EXPECT().GetSession(mock.Anything, sessionID).Return(tc.session, tc.sessionErr)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, store, false),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)
			addAuthorization(request, AuthTypeBearer, util.RandomString(24))

			router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}
