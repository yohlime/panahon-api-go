package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRateLimitTestRouter(t *testing.T) *DefaultRouter {
	t.Helper()

	gin.SetMode(gin.TestMode)
	config := util.Config{
		GinMode:             gin.TestMode,
		APIBasePath:         "/",
		SwagAPIBasePath:     "/",
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		FrontendBaseURL:     "http://localhost:8000",
		RateLimit: util.RateLimitConfig{
			Enabled:             true,
			LoginPerMinute:      2,
			RegistrationPerHour: 10,
			ResetPerMinute:      1,
			APIKeyPerMinute:     10,
		},
	}
	logger := util.NewLogger(config)
	handler := handlers.NewDefaultHandler(config, nil, nil, logger)

	return NewDefaultRouter(config, handler, nil, nil, logger)
}

func TestPublicAuthEndpointsRateLimited(t *testing.T) {
	router := newRateLimitTestRouter(t)

	post := func(path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "198.51.100.10:1234"
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	// Login: the first two malformed requests reach the handler (400), the
	// third is stopped by the limiter.
	require.Equal(t, http.StatusBadRequest, post("/users/login").Code)
	require.Equal(t, http.StatusBadRequest, post("/users/login").Code)
	limited := post("/users/login")
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	require.NotEmpty(t, limited.Header().Get("Retry-After"))

	// Password reset has its own bucket.
	require.Equal(t, http.StatusBadRequest, post("/users/password-reset/validate").Code)
	require.Equal(t, http.StatusTooManyRequests, post("/users/password-reset/validate").Code)
}
