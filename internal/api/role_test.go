package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mockdb "github.com/emiliogozo/panahon-api-go/db/mocks"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListRolesAPI(t *testing.T) {
	n := 10
	roles := make([]db.Role, n)
	for i := 0; i < n; i++ {
		roles[i] = randomRole(t)
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
				store.EXPECT().ListRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(roles, nil)
				store.EXPECT().CountRoles(mock.AnythingOfType("*gin.Context")).
					Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchRoles(t, recorder.Body, roles)
			},
		},
		{
			name: "InternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.Role{}, sql.ErrConnDone)
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
				store.AssertNotCalled(t, "ListRoles", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
				store.AssertNotCalled(t, "ListRoles", mock.AnythingOfType("*gin.Context"), mock.Anything)
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
				store.EXPECT().ListRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.Role{}, nil)
				store.EXPECT().CountRoles(mock.AnythingOfType("*gin.Context")).
					Return(int64(n), nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchRoles(t, recorder.Body, []db.Role{})
			},
		},
		{
			name: "CountInternalError",
			query: Query{
				Page:    1,
				PerPage: int32(n),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListRoles(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return([]db.Role{}, nil)
				store.EXPECT().CountRoles(mock.AnythingOfType("*gin.Context")).
					Return(int64(n), sql.ErrConnDone)
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/roles", server.config.APIBasePath)
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

func TestGetRoleAPI(t *testing.T) {
	role := randomRole(t)

	testCases := []struct {
		name          string
		roleID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			roleID: role.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRole(mock.AnythingOfType("*gin.Context"), role.ID).
					Return(role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchRole(t, recorder.Body, role)
			},
		},
		{
			name:   "NotFound",
			roleID: role.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRole(mock.AnythingOfType("*gin.Context"), role.ID).
					Return(db.Role{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			roleID: role.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRole(mock.AnythingOfType("*gin.Context"), role.ID).
					Return(db.Role{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "InvalidID",
			roleID: 0,
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "GetRole", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/roles/%d", server.config.APIBasePath, tc.roleID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestCreateRoleAPI(t *testing.T) {
	role := randomRole(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name: "OK",
			body: gin.H{
				"name":        role.Name,
				"description": role.Description,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchRole(t, recorder.Body, role)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"name":        role.Name,
				"description": role.Description,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Role{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "DuplicateName",
			body: gin.H{
				"name":        role.Name,
				"description": role.Description,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Role{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "InvalidName",
			body: gin.H{
				"name":        "invalid-role#1",
				"description": role.Description,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertNotCalled(t, "CreateRole", mock.AnythingOfType("*gin.Context"), mock.Anything)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(t)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/roles", server.config.APIBasePath)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestUpdateRoleAPI(t *testing.T) {
	role := randomRole(t)

	testCases := []struct {
		name          string
		roleID        int64
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			roleID: role.ID,
			body: gin.H{
				"id":          role.ID,
				"description": role.Description,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateRoleParams{
					ID:          role.ID,
					Description: role.Description,
				}

				store.EXPECT().UpdateRole(mock.AnythingOfType("*gin.Context"), arg).
					Return(role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchRole(t, recorder.Body, role)
			},
		},
		{
			name:   "InternalError",
			roleID: role.ID,
			body:   gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Role{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "RoleNotFound",
			roleID: role.ID,
			body:   gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
					Return(db.Role{}, db.ErrRecordNotFound)
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("%s/roles/%d", server.config.APIBasePath, tc.roleID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func TestDeleteRoleAPI(t *testing.T) {
	role := randomRole(t)

	testCases := []struct {
		name          string
		roleID        int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recoder *httptest.ResponseRecorder, store *mockdb.MockStore)
	}{
		{
			name:   "OK",
			roleID: role.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteRole(mock.AnythingOfType("*gin.Context"), role.ID).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder, store *mockdb.MockStore) {
				store.AssertExpectations(t)
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			roleID: role.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteRole(mock.AnythingOfType("*gin.Context"), mock.Anything).
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

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("%s/roles/%d", server.config.APIBasePath, tc.roleID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			tc.checkResponse(recorder, store)
		})
	}
}

func randomRole(t *testing.T) db.Role {
	return db.Role{
		ID:   util.RandomInt[int64](1, 1000),
		Name: strings.ToUpper(util.RandomString(12)),
		Description: pgtype.Text{
			String: util.RandomString(24),
			Valid:  true,
		},
	}
}

func requireBodyMatchRole(t *testing.T, body *bytes.Buffer, role db.Role) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotRole db.Role
	err = json.Unmarshal(data, &gotRole)

	require.NoError(t, err)
	require.Equal(t, role.Name, gotRole.Name)
	require.Equal(t, role.Description, gotRole.Description)
}

func requireBodyMatchRoles(t *testing.T, body *bytes.Buffer, roles []db.Role) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotRoles paginatedRoles
	err = json.Unmarshal(data, &gotRoles)

	require.NoError(t, err)
	for i, role := range roles {
		require.Equal(t, role.Name, gotRoles.Items[i].Name)
		require.Equal(t, role.Description.String, gotRoles.Items[i].Description)
	}
}
