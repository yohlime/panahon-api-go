package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/emiliogozo/panahon-api-go/db/mocks"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
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
		users[i], _ = randomUser(t)
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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/users", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to request URL
			q := request.URL.Query()
			q.Add("page", fmt.Sprintf("%d", tc.query.Page))
			q.Add("per_page", fmt.Sprintf("%d", tc.query.PerPage))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetUserAPI(t *testing.T) {
	user, _ := randomUser(t)

	n := 5
	roles := make([]db.Role, n)
	roleNames := make([]string, n)
	for i := 0; i < n; i++ {
		roles[i] = randomRole(t)
		roleNames[i] = roles[i].Name
	}

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
				requireBodyMatchUser(t, recorder.Body, newUser(user, nil))
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
				requireBodyMatchUser(t, recorder.Body, newUser(user, roleNames))
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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/users/%d", server.config.APIBasePath, tc.userID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	n := 5
	roles := make([]db.Role, n)
	roleNames := make([]string, n)
	withInvalidRoleNames := make([]string, n)
	userRoles := make([]db.UserRolesParams, n)
	var withInvalidRoleNamesRet []string
	var withInvalidRolesRet []db.UserRolesParams
	for i := 0; i < n; i++ {
		roles[i] = randomRole(t)
		roleName := roles[i].Name
		roleNames[i] = roleName
		userRoles[i] = db.UserRolesParams{
			Username: user.Username,
			RoleName: roleName,
		}
		withInvalidRoleNames[i] = roleName
		if i == 1 {
			withInvalidRoleNames[i] = "INVALID"
		}
		if i != 1 && i != 3 {
			withInvalidRolesRet = append(withInvalidRolesRet, userRoles[i])
			withInvalidRoleNamesRet = append(withInvalidRoleNamesRet, roleName)
		}
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
				requireBodyMatchUser(t, recorder.Body, newUser(user, nil))
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
					Return(userRoles, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, newUser(user, roleNames))
			},
		},
		{
			name: "WithInvalidRoles",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
				"roles":     withInvalidRoleNames,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(user, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(withInvalidRolesRet, []error{db.ErrRecordNotFound})
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, newUser(user, withInvalidRoleNamesRet))
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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/users", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestUpdateUserAPI(t *testing.T) {
	user, _ := randomUser(t)

	n := 5
	roles := make([]db.Role, n)
	roleNames := make([]string, n)
	for i := 0; i < n; i++ {
		roles[i] = randomRole(t)
		roleNames[i] = roles[i].Name
	}

	toAttachRolesNames := roleNames[0:3]
	attachedRolesNames := roleNames[2:4]
	addedUserRoles := []db.UserRolesParams{
		{
			Username: user.Username,
			RoleName: roleNames[0],
		},
		{
			Username: user.Username,
			RoleName: roleNames[1],
		},
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
				requireBodyMatchUser(t, recorder.Body, newUser(user, nil))
			},
		},
		{
			name:   "WithRoles",
			userID: user.ID,
			body: gin.H{
				"id":        user.ID,
				"username":  user.Username,
				"full_name": user.FullName,
				"roles":     toAttachRolesNames,
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
					Return(attachedRolesNames, nil)
				store.EXPECT().BulkCreateUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(addedUserRoles, nil)
				store.EXPECT().BulkDeleteUserRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, newUser(user, toAttachRolesNames))
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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/users/%d", server.config.APIBasePath, tc.userID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteUserAPI(t *testing.T) {
	user, _ := randomUser(t)

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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/users/%d", server.config.APIBasePath, tc.userID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestRegisterUserAPI(t *testing.T) {
	user, password := randomUser(t)

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

			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/users/register", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestLoginUserAPI(t *testing.T) {
	user, password := randomUser(t)

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
				tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything).Return(util.RandomString(32), &token.Payload{}, nil).Twice()
				store.EXPECT().CreateSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Session{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)

				cookies := recorder.Result().Cookies()
				require.Len(t, cookies, 2)

				for _, cookie := range cookies {
					require.Contains(t, []string{accessTokenCookieName, refreshTokenCookieName}, cookie.Name)
				}
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
				require.Empty(t, cookies)
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
				require.Empty(t, cookies)
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
				require.Empty(t, cookies)
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
				require.Empty(t, cookies)
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

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/users/login", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestGetAuthUserAPI(t *testing.T) {
	user, _ := randomUser(t)
	tokenStr := util.RandomString(32)

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
				addAuthorization(t, request, authTypeBearer, tokenStr)
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
				addAuthorization(t, request, authTypeBearer, tokenStr)
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

			server := newTestServer(t, store, tokenMaker)

			url := fmt.Sprintf("%s/users/testauth", server.config.APIBasePath)
			server.router.GET(
				url,
				authMiddleware(tokenMaker, false),
				server.GetAuthUser,
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder, store)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		ID:       util.RandomInt[int64](1, 1000),
		Username: util.RandomString(12),
		Password: hashedPassword,
		FullName: util.RandomString(24),
		Email:    util.RandomEmail(),
	}
	return
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser User
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
