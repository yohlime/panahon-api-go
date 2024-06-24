package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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
				addAuthorization(t, request, models.AuthTypeCookie, tokenStr)
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
				addAuthorization(t, request, "unsupported", "")
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationFormat",
			setupAuth: func(t *testing.T, request *http.Request) {
				authorizationHeader := fmt.Sprintf("%s%s", models.AuthTypeBearer, util.RandomString(12))
				request.Header.Set(models.AuthHeaderKey, authorizationHeader)
			},
			buildStubs: func(tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, request *http.Request) {
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, tc.permissive),
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router := gin.Default()
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, false),
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
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

			router := gin.Default()

			tc.buildStubs(tokenMaker)

			authPath := "/auth"
			router.GET(
				authPath,
				AuthMiddleware(tokenMaker, false),
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

func addAuthorization(
	t *testing.T,
	request *http.Request,
	authType string,
	token string,
) {
	if authType == models.AuthTypeBearer {
		authorizationHeader := fmt.Sprintf("%s %s", authType, token)
		request.Header.Set(models.AuthHeaderKey, authorizationHeader)
	} else if authType == models.AuthTypeCookie {
		addAccessTokenCookie(request, token)
	}
}

func addAccessTokenCookie(request *http.Request, token string) {
	request.AddCookie(&http.Cookie{Name: models.AccessTokenCookieName, Value: token})
}
