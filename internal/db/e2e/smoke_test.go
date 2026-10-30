package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/factory"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/routers"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

var (
	testStore  db.Store
	testConfig util.Config
)

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../..")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	config.GinMode = gin.ReleaseMode
	config.Environment = "test"
	config.EnableConsoleLogging = false
	config.EnableFileLogging = false
	testConfig = config

	pg := db.NewDockerPostgres(config)
	migrationPath := config.MigrationPath
	if !filepath.IsAbs(migrationPath) {
		migrationPath = filepath.Join("../../..", migrationPath)
	}

	migration, err := migrate.New("file://"+migrationPath, pg.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "construct migration: %v\n", err)
		pg.Close()
		os.Exit(1)
	}
	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "apply migrations: %v\n", err)
		_, _ = migration.Close()
		pg.Close()
		os.Exit(1)
	}

	testStore = db.NewStore(pg.Conn)

	code := m.Run()

	_, _ = migration.Close()
	pg.Close()
	os.Exit(code)
}

type apiClient struct {
	t      *testing.T
	base   string
	client *http.Client
}

func newAPIClient(t *testing.T, base string) *apiClient {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &apiClient{
		t:    t,
		base: base,
		client: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
	}
}

func (c *apiClient) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, c.base+path, reader)
	require.NoError(c.t, err)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.client.Do(request)
	require.NoError(c.t, err)
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	require.NoError(c.t, err)
	require.Equal(c.t, wantStatus, response.StatusCode, "%s %s: %s", method, path, data)

	if out != nil {
		require.NoError(c.t, json.Unmarshal(data, out), "response: %s", data)
	}
}

func TestSmokeFlow(t *testing.T) {
	ctx := context.Background()
	logger := util.NewLogger(testConfig)

	tokenMaker, err := token.NewPasetoMaker(testConfig.TokenSymmetricKey)
	require.NoError(t, err)

	handler := handlers.NewDefaultHandler(testConfig, testStore, tokenMaker, logger)
	router := routers.NewDefaultRouter(testConfig, handler, tokenMaker, testStore, logger)
	server := httptest.NewServer(router)
	defer server.Close()

	// The USER role must exist before the admin invites a user with that role.
	_, err = factory.EnsureRole(ctx, testStore, factory.RoleUser, factory.RoleDescriptions[factory.RoleUser])
	require.NoError(t, err)

	adminSpec := factory.UserSpec{
		Username: "e2eadmin",
		FullName: "E2E Admin",
		Email:    "e2eadmin@example.com",
		Password: "E2EAdminPass1",
		Roles:    []string{factory.RoleAdmin},
	}
	require.NoError(t, factory.SeedUsers(ctx, testStore, logger, []factory.UserSpec{adminSpec}))

	admin, err := testStore.GetUserByUsername(ctx, adminSpec.Username)
	require.NoError(t, err)

	baseURL := server.URL + strings.TrimRight(testConfig.APIBasePath, "/")
	adminClient := newAPIClient(t, baseURL)

	// 1. Admin authentication via the cookie session.
	adminClient.do(http.MethodPost, "/users/login", map[string]string{
		"username": adminSpec.Username,
		"password": adminSpec.Password,
	}, http.StatusNoContent, nil)

	// 2. Admin invitations and registration-link generation.
	inviteeUsername := "e2einvitee"
	inviteePassword := "E2EInviteePass1"
	var created struct {
		User            models.User `json:"user"`
		RegistrationURL string      `json:"registration_url"`
	}
	adminClient.do(http.MethodPost, "/users", map[string]any{
		"username":  inviteeUsername,
		"full_name": "Test Invitee",
		"roles":     []string{factory.RoleUser},
	}, http.StatusOK, &created)
	require.Equal(t, inviteeUsername, created.User.Username)
	require.Equal(t, models.UserStatusInvited, created.User.Status)
	require.Contains(t, created.User.Roles, factory.RoleUser)

	registrationURL, err := url.Parse(created.RegistrationURL)
	require.NoError(t, err)
	registrationToken := registrationURL.Query().Get("token")
	require.NotEmpty(t, registrationToken)

	// 3. The invitee validates and completes registration, then signs in.
	publicClient := newAPIClient(t, baseURL)
	var registrationInfo struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}
	publicClient.do(http.MethodPost, "/users/complete-registration/validate", map[string]string{
		"token": registrationToken,
	}, http.StatusOK, &registrationInfo)
	require.Equal(t, inviteeUsername, registrationInfo.Username)

	inviteeClient := newAPIClient(t, baseURL)
	inviteeClient.do(http.MethodPost, "/users/complete-registration/accept", map[string]string{
		"token":            registrationToken,
		"password":         inviteePassword,
		"confirm_password": inviteePassword,
	}, http.StatusNoContent, nil)

	var invitee models.User
	inviteeClient.do(http.MethodGet, "/users/auth", nil, http.StatusOK, &invitee)
	require.Equal(t, inviteeUsername, invitee.Username)
	require.Equal(t, models.UserStatusActive, invitee.Status)
	require.Contains(t, invitee.Roles, factory.RoleUser)

	loginClient := newAPIClient(t, baseURL)
	loginClient.do(http.MethodPost, "/users/login", map[string]string{
		"username": inviteeUsername,
		"password": inviteePassword,
	}, http.StatusNoContent, nil)
	var loggedIn models.User
	loginClient.do(http.MethodGet, "/users/auth", nil, http.StatusOK, &loggedIn)
	require.Equal(t, inviteeUsername, loggedIn.Username)

	// 4. Admin creates an API key and uses it against a gated endpoint.
	var tokenResponse models.APITokenWithSecret
	adminClient.do(http.MethodPost, fmt.Sprintf("/users/%d/api-tokens", admin.ID), map[string]string{
		"name": "e2e-smoke",
	}, http.StatusCreated, &tokenResponse)
	require.NotEmpty(t, tokenResponse.Token)

	anonymousClient := newAPIClient(t, baseURL)
	anonymousClient.do(http.MethodGet, "/observations?limit=1", nil, http.StatusUnauthorized, nil)

	apiKeyClient := newAPIClient(t, baseURL)
	apiKeyClient.do(http.MethodGet, "/observations?limit=1&api_key="+url.QueryEscape(tokenResponse.Token), nil, http.StatusOK, nil)
}
