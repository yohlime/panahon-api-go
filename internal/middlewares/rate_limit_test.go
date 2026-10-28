package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRateLimitRequest(ip string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.RemoteAddr = ip + ":1234"
	return request
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.POST("/login", RateLimit(ratelimit.New(), "login", 2, time.Minute), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	serve := func(ip string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, newRateLimitRequest(ip))
		return recorder
	}

	require.Equal(t, http.StatusNoContent, serve("192.0.2.1").Code)
	require.Equal(t, http.StatusNoContent, serve("192.0.2.1").Code)

	limited := serve("192.0.2.1")
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	retryAfter, err := strconv.Atoi(limited.Header().Get("Retry-After"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, retryAfter, 1)
	require.Contains(t, limited.Body.String(), "too many requests")

	// A different client address is not affected.
	require.Equal(t, http.StatusNoContent, serve("192.0.2.2").Code)
}

func TestRateLimitMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.POST("/login", RateLimit(ratelimit.New(), "login", 0, time.Minute), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	for range 5 {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, newRateLimitRequest("192.0.2.1"))
		require.Equal(t, http.StatusNoContent, recorder.Code)
	}
}

func TestAPIKeyRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/observations", APIKeyRateLimit(ratelimit.New(), 1, time.Minute), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	serve := func(ip, query string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/observations"+query, nil)
		request.RemoteAddr = ip + ":1234"
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	// Without an API key the middleware leaves the request alone.
	require.Equal(t, http.StatusNoContent, serve("192.0.2.1", "").Code)
	require.Equal(t, http.StatusNoContent, serve("192.0.2.1", "").Code)

	// API-key requests are limited before validation.
	require.Equal(t, http.StatusNoContent, serve("192.0.2.1", "?api_key=secret").Code)
	limited := serve("192.0.2.1", "?api_key=secret")
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	require.NotEmpty(t, limited.Header().Get("Retry-After"))

	// Other addresses still work.
	require.Equal(t, http.StatusNoContent, serve("192.0.2.2", "?api_key=secret").Code)
}
