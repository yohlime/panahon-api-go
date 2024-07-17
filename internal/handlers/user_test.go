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
	"github.com/emiliogozo/panahon-api-go/internal/middlewares"
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

func TestListUsersAPI(t *testing.T) {
	n := 10
	users := make([]db.User, n)
	for i := 0; i < n; i++ {
		users[i], _, _ = randomUser(t)
	}

	type Query struct {
		Page    int32
		PerPage int32
	}

	testCases := []struct {
		name          string
		query         Query
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(users, nil)
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context")).Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUsers(t, recorder.Body, users)
			},
		},
		{
			name: "InternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPage",
			query: Query{
				Page:    -1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListUsers", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidLimit",
			query: Query{
				Page:    1,
				PerPage: 10000,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "ListUsers", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "EmptySlice",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.User{}, nil)
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context")).Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUsers(t, recorder.Body, []db.User{})
			},
		},
		{
			name: "CountInternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.User{}, nil)
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context")).Return(0, sql.ErrConnDone)
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
			router.GET("/", handler.ListUsers)

			recorder := httptest.NewRecorder()

			url := "/"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			q := request.URL.Query()
			q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			q.Add("per_page", fmt.Sprintf("%d", tc.query.PerPage))
			request.URL.RawQuery = q.Encode()

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetUserAPI(t *testing.T) {
	user, _, roleNames := randomUser(t)

	testCases := []struct {
		name          string
		userID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, nil))
			},
		},
		{
			name:   "WithRoles",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(roleNames, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, roleNames))
			},
		},
		{
			name:   "NotFound",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "InvalidID",
			userID: 0,
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
			router.GET(":id", handler.GetUser)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d", tc.userID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestCreateUserAPI(t *testing.T) {
	user, password, roleNames := randomUser(t)

	var userRoleParams []db.UserRolesParams
	for _, r := range roleNames {
		userRoleParams = append(userRoleParams, db.UserRolesParams{
			Username: user.Username,
			RoleName: r,
		})
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, nil))
			},
		},
		{
			name: "WithRoles",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
				"roles":     roleNames,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(user, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(userRoleParams, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, roleNames))
			},
		},
		{
			name: "WithInvalidRoles",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
				"roles":     roleNames,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(user, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.UserRolesParams{}, []error{db.ErrRecordNotFound})
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, []string{}))
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username":  "invalid-user#1",
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     "invalid-email",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "TooShortPassword",
			body: gin.H{
				"username":  user.Username,
				"password":  "123",
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
			router.POST("", handler.CreateUser)

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

func TestUpdateUserAPI(t *testing.T) {
	user, _, _ := randomUser(t)
	newRoleNames := []string{"ADMIN", "SUPERADMIN"}

	var addedUserRoles []db.UserRolesParams
	for _, r := range newRoleNames {
		addedUserRoles = append(addedUserRoles, db.UserRolesParams{
			Username: user.Username,
			RoleName: r,
		})
	}

	testCases := []struct {
		name          string
		userID        int64
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			userID: user.ID,
			body: gin.H{
				"id":        user.ID,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					ID: user.ID,
					FullName: pgtype.Text{
						String: user.FullName,
						Valid:  true,
					},
				}

				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), arg).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, nil))
			},
		},
		{
			name:   "WithRoles",
			userID: user.ID,
			body: gin.H{
				"id":        user.ID,
				"username":  user.Username,
				"full_name": user.FullName,
				"roles":     newRoleNames,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					ID: user.ID,
					FullName: pgtype.Text{
						String: user.FullName,
						Valid:  true,
					},
				}

				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), arg).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
					Return([]string{}, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(addedUserRoles, nil)
				store.EXPECT().BulkDeleteUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, models.NewUser(user, newRoleNames))
			},
		},
		{
			name:   "InternalError",
			userID: user.ID,
			body:   gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "UserNotFound",
			userID: user.ID,
			body:   gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrRecordNotFound)
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
			router.PUT(":id", handler.UpdateUser)

			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/%d", tc.userID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteUserAPI(t *testing.T) {
	user, _, _ := randomUser(t)

	testCases := []struct {
		name          string
		userID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteUser(mock.AnythingOfType("*gin.Context"), user.ID).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			userID: user.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(sql.ErrConnDone)
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
			router.DELETE(":id", handler.DeleteUser)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/%d", tc.userID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestRegisterUserAPI(t *testing.T) {
	user, password, _ := randomUser(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":         user.Username,
				"password":         password,
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(user, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.UserRolesParams{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username":         user.Username,
				"password":         password,
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: gin.H{
				"username":         user.Username,
				"password":         password,
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username":         "invalid-user#1",
				"password":         password,
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"username":         user.Username,
				"password":         password,
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            "invalid-email",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "TooShortPassword",
			body: gin.H{
				"username":         user.Username,
				"password":         "123",
				"confirm_password": password,
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MismatchPassword",
			body: gin.H{
				"username":         user.Username,
				"password":         password,
				"confirm_password": "mismatchpass",
				"full_name":        user.FullName,
				"email":            user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateUser", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
			router.POST("/register", handler.RegisterUser)

			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/register"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestLoginUserAPI(t *testing.T) {
	expectedCookieNames := []string{models.AccessTokenCookieName, models.RefreshTokenCookieName}
	user, password, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)
	tokenExpiresAt := time.Now().Add(6 * time.Hour)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]string{}, nil)
				tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything).Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil).Twice()
				store.EXPECT().CreateSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Session{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				require.GreaterOrEqual(t, len(cookies), 2)

				var foundCookieCount int
				for _, cookie := range cookies {
					for _, cName := range expectedCookieNames {
						if cookie.Name == cName {
							require.NotEmpty(t, cookie.Value)
							require.Greater(t, cookie.MaxAge, 0)
							foundCookieCount++
							break
						}
					}
				}
				require.Equal(t, 2, foundCookieCount)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"username": "NotFound",
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)

				cookies := recorder.Result().Cookies()
				var foundCookies []*http.Cookie
				for _, cookie := range cookies {
					for _, cookieName := range expectedCookieNames {
						if cookie.Name == cookieName {
							foundCookies = append(foundCookies, cookie)
							break
						}
					}
					if len(foundCookies) == len(expectedCookieNames) {
						break
					}
				}
				require.Len(t, foundCookies, 0)
			},
		},
		{
			name: "IncorrectPassword",
			body: gin.H{
				"username": user.Username,
				"password": "incorrect",
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				cookies := recorder.Result().Cookies()
				var foundCookies []*http.Cookie
				for _, cookie := range cookies {
					for _, cookieName := range expectedCookieNames {
						if cookie.Name == cookieName {
							foundCookies = append(foundCookies, cookie)
							break
						}
					}
					if len(foundCookies) == len(expectedCookieNames) {
						break
					}
				}
				require.Len(t, foundCookies, 0)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)

				cookies := recorder.Result().Cookies()
				var foundCookies []*http.Cookie
				for _, cookie := range cookies {
					for _, cookieName := range expectedCookieNames {
						if cookie.Name == cookieName {
							foundCookies = append(foundCookies, cookie)
							break
						}
					}
					if len(foundCookies) == len(expectedCookieNames) {
						break
					}
				}
				require.Len(t, foundCookies, 0)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username":  "invalid-user#1",
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetUserByUsername", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)

				cookies := recorder.Result().Cookies()
				var foundCookies []*http.Cookie
				for _, cookie := range cookies {
					for _, cookieName := range expectedCookieNames {
						if cookie.Name == cookieName {
							foundCookies = append(foundCookies, cookie)
							break
						}
					}
					if len(foundCookies) == len(expectedCookieNames) {
						break
					}
				}
				require.Len(t, foundCookies, 0)
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
			router.POST("/login", handler.LoginUser)

			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/login"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestLogoutUserAPI(t *testing.T) {
	expectedCookieNames := []string{models.AccessTokenCookieName, models.RefreshTokenCookieName}
	tokenStr := gofakeit.LetterN(32)

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{}, nil)
				store.EXPECT().DeleteSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					for _, cName := range expectedCookieNames {
						if cookie.Name == cName {
							require.Empty(t, cookie.Value)
							require.Less(t, cookie.MaxAge, 0)
							break
						}
					}
				}
			},
		},
		{
			name:      "NoRefreshCookie",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{}, token.ErrInvalidToken)
				store.EXPECT().DeleteSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					for _, cName := range expectedCookieNames {
						if cookie.Name == cName {
							require.Empty(t, cookie.Value)
							require.Less(t, cookie.MaxAge, 0)
							break
						}
					}
				}
			},
		},
		{
			name: "InvalidRefreshToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{}, token.ErrInvalidToken)
				store.EXPECT().DeleteSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					for _, cName := range expectedCookieNames {
						if cookie.Name == cName {
							require.Empty(t, cookie.Value)
							require.Less(t, cookie.MaxAge, 0)
							break
						}
					}
				}
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addRefreshTokenCookie(request, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{}, nil)
				store.EXPECT().DeleteSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				for _, cookie := range cookies {
					for _, cName := range expectedCookieNames {
						if cookie.Name == cName {
							require.Empty(t, cookie.Value)
							require.Less(t, cookie.MaxAge, 0)
							break
						}
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

			handler := newTestHandler(store, tokenMaker)

			router := gin.Default()
			router.POST("/logout", handler.LogoutUser)

			recorder := httptest.NewRecorder()

			url := "/logout"
			request, err := http.NewRequest(http.MethodPost, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, handler.tokenMaker)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetAuthUserAPI(t *testing.T) {
	user, _, _ := randomUser(t)
	tokenStr := gofakeit.LetterN(32)

	authUser := token.User{
		Username: user.Username,
	}

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, models.AuthTypeBearer, tokenStr)
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				tokenMaker.EXPECT().VerifyToken(mock.AnythingOfType("string")).Return(&token.Payload{User: authUser}, nil)
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.User{}, sql.ErrConnDone)
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
			tc.buildStubs(store, tokenMaker)

			handler := newTestHandler(store, tokenMaker)

			router := gin.Default()

			url := "/testauth"
			router.GET(
				url,
				middlewares.AuthMiddleware(tokenMaker, false),
				handler.GetAuthUser,
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, handler.tokenMaker)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string, roleNames []string) {
	password = gofakeit.Password(true, true, false, false, false, 12)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	var u models.User
	err = gofakeit.Struct(&u)
	require.NoError(t, err)

	user = db.User{
		ID:       int64(gofakeit.Number(1, 1000)),
		Username: u.Username,
		Password: hashedPassword,
		FullName: u.FullName,
		Email:    u.Email,
	}
	roleNames = u.Roles
	return
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user models.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser models.User
	err = json.Unmarshal(data, &gotUser)

	require.NoError(t, err)
	require.Equal(t, user.Username, gotUser.Username)
	require.Equal(t, user.FullName, gotUser.FullName)
	require.Equal(t, user.Email, gotUser.Email)
	require.Len(t, gotUser.Roles, len(user.Roles))

	for _, gotRoleName := range gotUser.Roles {
		containsRoleName := false
		for _, roleName := range user.Roles {
			if roleName == gotRoleName {
				containsRoleName = true
				break
			}
		}
		require.True(t, containsRoleName)
	}
}

func requireBodyMatchUsers(t *testing.T, body *bytes.Buffer, users []db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUsers paginatedUsers
	err = json.Unmarshal(data, &gotUsers)

	require.NoError(t, err)
	for i, user := range users {
		require.Equal(t, user.Username, gotUsers.Items[i].Username)
		require.Equal(t, user.FullName, gotUsers.Items[i].FullName)
		require.Equal(t, user.Email, gotUsers.Items[i].Email)
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
