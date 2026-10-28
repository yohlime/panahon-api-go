package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newHealthContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	return ctx, recorder
}

func TestHealthzAPI(t *testing.T) {
	store := mockdb.NewMockStore(t)
	handler := newTestHandler(store, nil)
	ctx, recorder := newHealthContext(t)

	handler.Healthz(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

	var response healthRes
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "ok", response.Status)
}

func TestReadyzAPI(t *testing.T) {
	t.Run("Ready", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().Ping(mock.Anything).Return(nil)

		handler := newTestHandler(store, nil)
		ctx, recorder := newHealthContext(t)

		handler.Readyz(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

		var response healthRes
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, "ok", response.Status)
	})

	t.Run("NilStore", func(t *testing.T) {
		handler := newTestHandler(nil, nil)
		ctx, recorder := newHealthContext(t)

		handler.Readyz(ctx)

		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		store.EXPECT().Ping(mock.Anything).Return(errors.New("db down"))

		handler := newTestHandler(store, nil)
		ctx, recorder := newHealthContext(t)

		handler.Readyz(ctx)

		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

		var response healthRes
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, "unavailable", response.Status)
	})
}
