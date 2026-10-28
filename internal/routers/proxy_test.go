package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newProxyTestRouter(t *testing.T, trustedProxies string) *DefaultRouter {
	t.Helper()

	gin.SetMode(gin.TestMode)
	config := util.Config{
		GinMode:             gin.TestMode,
		APIBasePath:         "/",
		SwagAPIBasePath:     "/",
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		FrontendBaseURL:     "http://localhost:8000",
		TrustedProxies:      trustedProxies,
	}
	logger := util.NewLogger(config)
	handler := handlers.NewDefaultHandler(config, nil, nil, logger)

	router := NewDefaultRouter(config, handler, nil, nil, logger)
	router.GET("/client-ip", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, ctx.ClientIP())
	})
	return router
}

func TestTrustedProxies(t *testing.T) {
	request := func(router *DefaultRouter) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
		req.RemoteAddr = "198.51.100.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}

	t.Run("ForwardedHeaderIgnoredByDefault", func(t *testing.T) {
		recorder := request(newProxyTestRouter(t, ""))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "198.51.100.1", recorder.Body.String())
	})

	t.Run("ForwardedHeaderFromTrustedProxy", func(t *testing.T) {
		recorder := request(newProxyTestRouter(t, "198.51.100.1"))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "203.0.113.9", recorder.Body.String())
	})
}
