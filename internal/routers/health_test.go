package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHealthTestRouter(t *testing.T) *DefaultRouter {
	t.Helper()

	gin.SetMode(gin.TestMode)
	config := util.Config{
		GinMode:             gin.TestMode,
		APIBasePath:         "/api/v1",
		SwagAPIBasePath:     "/api/v1",
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		FrontendBaseURL:     "http://localhost:8000",
	}
	logger := util.NewLogger(config)
	store := mockdb.NewMockStore(t)
	store.EXPECT().Ping(mock.Anything).Return(nil).Once()
	handler := handlers.NewDefaultHandler(config, store, nil, logger)

	return NewDefaultRouter(config, handler, nil, store, logger)
}

func TestHealthEndpoints(t *testing.T) {
	router := newHealthTestRouter(t)

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, health.Code)

	ready := httptest.NewRecorder()
	router.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, ready.Code)

	// Probes are not nested under the API prefix.
	prefixed := httptest.NewRecorder()
	router.ServeHTTP(prefixed, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))
	require.Equal(t, http.StatusNotFound, prefixed.Code)
}
