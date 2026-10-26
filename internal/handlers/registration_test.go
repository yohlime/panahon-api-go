package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func invitedUser(t *testing.T) db.User {
	user, _, _ := randomUser(t)
	user.Status = models.UserStatusInvited
	user.Email = pgtype.Text{String: "invited@example.com", Valid: true}
	return user
}

func TestValidateCompleteRegistrationAPI(t *testing.T) {
	user := invitedUser(t)
	rawToken := "raw-registration-token"

	validToken := db.UserToken{
		UserID:    user.ID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.GetUserTokenByHashParams) bool {
					return arg.TokenHash == util.HashToken(rawToken) && arg.Purpose == db.UserTokenPurposeRegistration
				})).Return(validToken, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var got registrationInfoRes
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
				require.Equal(t, user.Username, got.Username)
				require.Equal(t, user.FullName, got.FullName)
				require.Equal(t, user.Email.String, got.Email)
			},
		},
		{
			name: "InvalidToken",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.UserToken{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ExpiredToken",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				expired := validToken
				expired.ExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(expired, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UsedToken",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				used := validToken
				used.UsedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(used, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "UserAlreadyActive",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				active := user
				active.Status = models.UserStatusActive
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(validToken, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(active, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "MissingToken",
			body:       gin.H{},
			buildStubs: func(store *mockdb.MockStore) {},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{"token": rawToken},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.UserToken{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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
			router.POST("/users/complete-registration/validate", handler.ValidateCompleteRegistration)

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodPost, "/users/complete-registration/validate", bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
			store.AssertExpectations(t)
		})
	}
}

func TestCompleteRegistrationAPI(t *testing.T) {
	user := invitedUser(t)
	activated := user
	activated.Status = models.UserStatusActive
	activated.Email = pgtype.Text{String: "new@example.com", Valid: true}
	activated.FullName = "New Name"

	rawToken := "raw-registration-token"
	tokenStr := gofakeit.LetterN(32)
	tokenExpiresAt := time.Now().Add(6 * time.Hour)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"token":            rawToken,
				"full_name":        "New Name",
				"email":            "NEW@example.com",
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().CompleteRegistrationTx(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CompleteRegistrationTxParams) bool {
					return arg.TokenHash == util.HashToken(rawToken) &&
						arg.SetEmail && arg.Email.Valid && arg.Email.String == "new@example.com" &&
						arg.FullName.Valid && arg.FullName.String == "New Name"
				})).Return(activated, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), activated.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().CreateSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Session{}, nil)
				tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything, mock.Anything).
					Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil).Twice()
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, recorder.Code)
				require.Len(t, recorder.Result().Cookies(), 2)
			},
		},
		{
			name: "EmptyEmail",
			body: gin.H{
				"token":            rawToken,
				"email":            "",
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().CompleteRegistrationTx(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CompleteRegistrationTxParams) bool {
					return arg.SetEmail && !arg.Email.Valid
				})).Return(activated, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), activated.ID).
					Return([]string{"USER"}, nil)
				store.EXPECT().CreateSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Session{}, nil)
				tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything, mock.Anything).
					Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil).Twice()
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"token":            rawToken,
				"email":            "not-an-email",
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "PasswordMismatch",
			body: gin.H{
				"token":            rawToken,
				"password":         "password123",
				"confirm_password": "different",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidToken",
			body: gin.H{
				"token":            rawToken,
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().CompleteRegistrationTx(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "DuplicateEmail",
			body: gin.H{
				"token":            rawToken,
				"email":            "taken@example.com",
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().CompleteRegistrationTx(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"token":            rawToken,
				"password":         "password123",
				"confirm_password": "password123",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().CompleteRegistrationTx(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tokenMaker := mocktoken.NewMockMaker(t)
			tc.buildStubs(store, tokenMaker)

			handler := newTestHandler(store, tokenMaker)

			router := gin.Default()
			router.POST("/users/complete-registration/accept", handler.CompleteRegistration)

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodPost, "/users/complete-registration/accept", bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
			store.AssertExpectations(t)
		})
	}
}

func TestCreateRegistrationLinkAPI(t *testing.T) {
	user := invitedUser(t)

	testCases := []struct {
		name          string
		userID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(user, nil)
				store.EXPECT().CreateRegistrationTokenTx(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateRegistrationTokenTxParams) bool {
					return arg.UserID == user.ID && len(arg.TokenHash) == 64
				})).Return(db.UserToken{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var got registrationLinkRes
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
				require.Contains(t, got.RegistrationURL, "http://localhost:8000/complete-registration?token=")
			},
		},
		{
			name:   "UserNotInvited",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				active := user
				active.Status = models.UserStatusActive
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(active, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name:   "UserNotFound",
			userID: 999,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), int64(999)).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "TokenCreationError",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).Return(user, nil)
				store.EXPECT().CreateRegistrationTokenTx(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.UserToken{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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
			router.POST("/users/:id/registration-link", handler.CreateRegistrationLink)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodPost, "/users/"+strconv.FormatInt(tc.userID, 10)+"/registration-link", nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
			store.AssertExpectations(t)
		})
	}
}

func TestLoginInvitedUser(t *testing.T) {
	user, password, _ := randomUser(t)
	user.Status = models.UserStatusInvited

	store := mockdb.NewMockStore(t)
	store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
		Return(user, nil)

	handler := newTestHandler(store, nil)

	router := gin.Default()
	router.POST("/login", handler.LoginUser)

	data, err := json.Marshal(gin.H{"username": user.Username, "password": password})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(data))
	require.NoError(t, err)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Empty(t, recorder.Result().Cookies())
	store.AssertExpectations(t)
}
