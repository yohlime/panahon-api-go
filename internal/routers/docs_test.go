package routers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiDocs "github.com/emiliogozo/panahon-api-go/internal/docs/api"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDocsTestRouter(t *testing.T, swagBasePath ...string) *DefaultRouter {
	t.Helper()

	basePath := "/api/v1"
	if len(swagBasePath) > 0 {
		basePath = swagBasePath[0]
	}

	gin.SetMode(gin.TestMode)
	config := util.Config{
		GinMode:                   gin.TestMode,
		APIBasePath:               "/api/v1",
		SwagAPIBasePath:           basePath,
		TokenSymmetricKey:         util.RandomString(32),
		AccessTokenDuration:       time.Minute,
		RefreshTokenDuration:      time.Hour,
		RegistrationTokenDuration: time.Hour,
		FrontendBaseURL:           "http://localhost:8000",
	}
	apiDocs.SwaggerInfo.BasePath = config.SwagAPIBasePath
	logger := util.NewLogger(config)
	handler := handlers.NewDefaultHandler(config, nil, nil, logger)

	return NewDefaultRouter(config, handler, nil, nil, logger)
}

func serveDocsRequest(router *DefaultRouter, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestDocsRootRedirects(t *testing.T) {
	router := newDocsTestRouter(t)

	for path, location := range map[string]string{
		"/api/v1/docs":         "/api/v1/docs/index.html",
		"/api/v1/docs/":        "/api/v1/docs/index.html",
		"/api/v1/public-docs":  "/api/v1/public-docs/index.html",
		"/api/v1/public-docs/": "/api/v1/public-docs/index.html",
	} {
		recorder := serveDocsRequest(router, http.MethodGet, path)
		require.Equal(t, http.StatusFound, recorder.Code, path)
		require.Equal(t, location, recorder.Header().Get("Location"), path)
	}
}

func TestDocsRedirectsUseExternalBasePath(t *testing.T) {
	router := newDocsTestRouter(t, "/data/api/v1")

	for path, location := range map[string]string{
		"/api/v1/docs":         "/data/api/v1/docs/index.html",
		"/api/v1/public-docs":  "/data/api/v1/public-docs/index.html",
		"/api/v1/public-docs/": "/data/api/v1/public-docs/index.html",
	} {
		recorder := serveDocsRequest(router, http.MethodGet, path)
		require.Equal(t, http.StatusFound, recorder.Code, path)
		require.Equal(t, location, recorder.Header().Get("Location"), path)
	}
}

func TestSwaggerUIPages(t *testing.T) {
	router := newDocsTestRouter(t)

	full := serveDocsRequest(router, http.MethodGet, "/api/v1/docs/index.html")
	require.Equal(t, http.StatusOK, full.Code)
	require.Contains(t, full.Body.String(), "Swagger UI")

	public := serveDocsRequest(router, http.MethodGet, "/api/v1/public-docs/index.html")
	require.Equal(t, http.StatusOK, public.Code)
	require.Contains(t, public.Body.String(), "Panahon Public API")
	require.Contains(t, public.Body.String(), "public-openapi.json")
}

func TestPublicOpenAPISpec(t *testing.T) {
	router := newDocsTestRouter(t)

	recorder := serveDocsRequest(router, http.MethodGet, "/api/v1/public-openapi.json")
	require.Equal(t, http.StatusOK, recorder.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &spec))

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)

	expected := []string{
		"/stations",
		"/stations/{station_id}",
		"/stations/nearest/observations/latest",
		"/stations/{station_id}/observations",
		"/stations/{station_id}/observations/{id}",
		"/stations/{station_id}/observations/latest",
		"/observations",
		"/observations/latest",
	}
	require.Len(t, paths, len(expected))
	for _, path := range expected {
		require.Contains(t, paths, path)
	}

	require.NotContains(t, paths, "/stations/stats")
	require.NotContains(t, paths, "/users")
	require.NotContains(t, paths, "/roles")

	definitions, ok := spec["definitions"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, definitions, "Station")
	require.Contains(t, definitions, "LatestObservation")
	require.NotContains(t, definitions, "User")
	require.NotContains(t, definitions, "Role")
	require.NotContains(t, definitions, "APIToken")

	securityDefinitions, ok := spec["securityDefinitions"].(map[string]any)
	require.True(t, ok)
	require.Len(t, securityDefinitions, 1)
	require.Contains(t, securityDefinitions, "APITokenAuth")

	stations := paths["/stations"].(map[string]any)["get"].(map[string]any)
	require.NotContains(t, stations, "security")

	observations := paths["/observations"].(map[string]any)["get"].(map[string]any)
	require.Equal(t, []any{map[string]any{"APITokenAuth": []any{}}}, observations["security"])

	info := spec["info"].(map[string]any)
	require.Equal(t, "Panahon Public API", info["title"])
}
