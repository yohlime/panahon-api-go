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
	for i := range n {
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
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).Return(int64(n), nil)
				store.EXPECT().ListUserRolesByUserIDs(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.ListUserRolesByUserIDsRow{}, nil)
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
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).Return(int64(n), nil)
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
				store.EXPECT().CountUsers(mock.AnythingOfType("*gin.Context"), mock.Anything).Return(0, sql.ErrConnDone)
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
			router.POST("", authAsCaller(nil), handler.CreateUser)

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
			router.PUT(":id", authAsCaller(nil), handler.UpdateUser)

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
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreateUserParams) bool {
					return arg.Status == models.UserStatusPending
				})).Return(user, nil)
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
				tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything, mock.Anything).Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil).Twice()
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
			name: "PendingUser",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				pending := user
				pending.Status = models.UserStatusPending
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(pending, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "RejectedUser",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker) {
				rejected := user
				rejected.Status = models.UserStatusRejected
				store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), user.Username).
					Return(rejected, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
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
		setupAuth     func(request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore, tokenMaker *mocktoken.MockMaker)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
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
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				addAuthorization(request, AuthTypeBearer, tokenStr)
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
				middlewares.AuthMiddleware(tokenMaker, store, false),
				handler.GetAuthUser,
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(request, handler.tokenMaker)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

// authAsCaller injects an authenticated payload for the given caller roles
// (defaulting to SUPERADMIN) so handler-level authorization can be exercised
// without going through the real auth middleware.
func authAsCaller(roles []string) gin.HandlerFunc {
	if len(roles) == 0 {
		roles = []string{string(models.SuperAdminRole)}
	}
	return func(ctx *gin.Context) {
		ctx.Set(models.AuthPayloadKey, &token.Payload{User: token.User{Username: "caller", Roles: roles}})
		ctx.Next()
	}
}

func TestCreateUserAdminCannotGrantSuperAdmin(t *testing.T) {
	user, password, _ := randomUser(t)

	store := mockdb.NewMockStore(t)
	handler := newTestHandler(store, nil)

	router := gin.Default()
	router.POST("", authAsCaller([]string{string(models.AdminRole)}), handler.CreateUser)

	body := gin.H{
		"username":  user.Username,
		"password":  password,
		"full_name": user.FullName,
		"email":     user.Email,
		"roles":     []string{"ADMIN", "SUPERADMIN"},
	}
	data, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	require.NoError(t, err)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	store.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestUpdateUserAdminCannotGrantSuperAdmin(t *testing.T) {
	user, _, _ := randomUser(t)

	store := mockdb.NewMockStore(t)
	handler := newTestHandler(store, nil)

	router := gin.Default()
	router.PUT(":id", authAsCaller([]string{string(models.AdminRole)}), handler.UpdateUser)

	body := gin.H{
		"id":        user.ID,
		"full_name": user.FullName,
		"roles":     []string{"SUPERADMIN"},
	}
	data, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/%d", user.ID), bytes.NewReader(data))
	require.NoError(t, err)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	store.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
}

func TestUpdateUserAdminCannotChangeSuperAdminRoles(t *testing.T) {
	user, _, _ := randomUser(t)

	store := mockdb.NewMockStore(t)
	store.EXPECT().ListUserRoles(mock.Anything, user.ID).Return([]string{string(models.SuperAdminRole)}, nil)

	handler := newTestHandler(store, nil)

	router := gin.Default()
	router.PUT(":id", authAsCaller([]string{string(models.AdminRole)}), handler.UpdateUser)

	body := gin.H{
		"id":        user.ID,
		"full_name": user.FullName,
		"roles":     []string{"USER"},
	}
	data, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/%d", user.ID), bytes.NewReader(data))
	require.NoError(t, err)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	store.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
}

func TestReviewUserAPI(t *testing.T) {
	reviewer := db.User{ID: 42, Username: "caller"}
	pending := db.User{ID: 7, Username: "newbie", Status: models.UserStatusPending}
	active := pending
	active.Status = models.UserStatusActive

	testCases := []struct {
		name       string
		path       string
		url        string
		fn         func(*DefaultHandler) gin.HandlerFunc
		buildStubs func(store *mockdb.MockStore)
		wantStatus int
	}{
		{
			name: "Approve",
			path: "/users/:id/approve",
			url:  "/users/7/approve",
			fn:   func(h *DefaultHandler) gin.HandlerFunc { return h.ApproveUser },
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, reviewer.Username).Return(reviewer, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), pending.ID).Return(pending, nil)
				store.EXPECT().ReviewUser(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.ReviewUserParams) bool {
					return arg.Status == models.UserStatusActive && arg.ReviewedBy.Int64 == reviewer.ID
				})).Return(active, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), pending.ID).Return([]string{"USER"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Reject",
			path: "/users/:id/reject",
			url:  "/users/7/reject",
			fn:   func(h *DefaultHandler) gin.HandlerFunc { return h.RejectUser },
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, reviewer.Username).Return(reviewer, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), pending.ID).Return(pending, nil)
				store.EXPECT().ReviewUser(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.ReviewUserParams) bool {
					return arg.Status == models.UserStatusRejected
				})).Return(active, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), pending.ID).Return([]string{"USER"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "NotPending",
			path: "/users/:id/approve",
			url:  "/users/7/approve",
			fn:   func(h *DefaultHandler) gin.HandlerFunc { return h.ApproveUser },
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, reviewer.Username).Return(reviewer, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), pending.ID).Return(active, nil)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "NotFound",
			path: "/users/:id/approve",
			url:  "/users/7/approve",
			fn:   func(h *DefaultHandler) gin.HandlerFunc { return h.ApproveUser },
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, reviewer.Username).Return(reviewer, nil)
				store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), pending.ID).Return(db.User{}, db.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)
			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.POST(tc.path, authAsCaller([]string{string(models.AdminRole)}), tc.fn(handler))

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodPost, tc.url, nil)
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}

func TestUpdateProfileAPI(t *testing.T) {
	user, password, _ := randomUser(t)

	testCases := []struct {
		name       string
		userID     int64
		body       gin.H
		buildStubs func(store *mockdb.MockStore)
		wantStatus int
	}{
		{
			name:   "OK",
			userID: user.ID,
			body: gin.H{
				"current_password": password,
				"email":            "updated@example.com",
			},
			buildStubs: func(store *mockdb.MockStore) {
				updated := user
				updated.Email = "updated@example.com"
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(user, nil)
				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.UpdateUserParams) bool {
					return arg.Email.Valid && arg.Email.String == "updated@example.com"
				})).Return(updated, nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).Return([]string{"USER"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "WrongCurrentPassword",
			userID: user.ID,
			body: gin.H{
				"current_password": "wrongpass",
				"email":            "updated@example.com",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(user, nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "CannotEditOtherUser",
			userID: user.ID + 1,
			body: gin.H{
				"current_password": password,
				"email":            "updated@example.com",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(user, nil)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "UsernameChanged",
			userID: user.ID,
			body: gin.H{
				"current_password": password,
				"username":         "newcaller",
			},
			buildStubs: func(store *mockdb.MockStore) {
				updated := user
				updated.Username = "newcaller"
				store.EXPECT().GetUserByUsername(mock.Anything, "caller").Return(user, nil)
				store.EXPECT().UpdateUser(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.UpdateUserParams) bool {
					return arg.Username.Valid && arg.Username.String == "newcaller"
				})).Return(updated, nil)
				store.EXPECT().DeleteSessionsByUserID(mock.AnythingOfType("*gin.Context"), user.ID).Return(nil)
				store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).Return([]string{"USER"}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)
			handler := newTestHandler(store, nil)

			router := gin.Default()
			router.PUT("/users/:id/profile", authAsCaller([]string{"USER"}), handler.UpdateProfile)

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/users/%d/profile", tc.userID), bytes.NewReader(data))
			require.NoError(t, err)

			router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)

			if tc.name == "UsernameChanged" {
				require.Len(t, recorder.Result().Cookies(), 2)
			}
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
		Status:   models.UserStatusActive,
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
