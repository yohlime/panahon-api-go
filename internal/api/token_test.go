package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRenewAccessToken(t *testing.T) {
	tokenStr := util.RandomString(24)
	tokenExpiresAt := time.Now().Add(6 * time.Hour)
	user, _ := randomUser(t)
	refreshPayload := token.Payload{User: token.User{Username: user.Username}}
	testCases := []struct {
		name          string
		body          gin.H
		setupToken    func(request *http.Request)
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
	}{
		{
			name: "ValidRefreshTokenCookie",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: tokenExpiresAt, Valid: true}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(user, nil)
				tokenMaker.EXPECT().CreateToken(mock.AnythingOfType("token.User"), mock.AnythingOfType("time.Duration")).
					Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				require.GreaterOrEqual(t, len(cookies), 1)

				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.NotEmpty(t, cookie.Value)
						require.Greater(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "ValidRefreshTokenFromRequest",
			body: gin.H{
				"refresh_token": tokenStr,
			},
			setupToken: func(request *http.Request) {},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: tokenExpiresAt, Valid: true}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(user, nil)
				tokenMaker.EXPECT().CreateToken(mock.AnythingOfType("token.User"), mock.AnythingOfType("time.Duration")).
					Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				require.GreaterOrEqual(t, len(cookies), 1)

				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.NotEmpty(t, cookie.Value)
						require.Greater(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "InvalidRefreshToken",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(nil, token.ErrInvalidToken)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "SessionNoEntry",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "SessionInternalError",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "SessionIsBlocked",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, IsBlocked: true, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "NoLinkedUser",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.User{}, pgx.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "UserInternalError",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "MismatchedUser",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(db.User{Username: "mismatched"}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "MismatchedToken",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: "tokenMismatched", ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "ExpiredToken",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * -6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
		{
			name: "CreateTokenInternalError",
			body: gin.H{},
			setupToken: func(request *http.Request) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.Anything).Return(&refreshPayload, nil)
				store.EXPECT().GetSession(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("uuid.UUID")).
					Return(db.Session{RefreshToken: tokenStr, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour * 6)}}, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("int64")).
					Return(user, nil)
				tokenMaker.EXPECT().CreateToken(mock.AnythingOfType("token.User"), mock.AnythingOfType("time.Duration")).
					Return("", &token.Payload{}, errors.New("error token create"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.AssertExpectations(t)
				tokenMaker.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					if cookie.Name == accessTokenCookieName {
						require.Empty(t, cookie.Value)
						require.Less(t, cookie.MaxAge, 0)
						break
					}
				}
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(store, tokenMaker)

			server := newTestServer(t, store, tokenMaker)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/tokens/renew", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupToken(request)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store, tokenMaker)
		})
	}
}

func addAccessTokenCookie(request *http.Request, token string) {
	request.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: token})
}

func addRefreshTokenCookie(request *http.Request, token string) {
	request.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: token})
}
