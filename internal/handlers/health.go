package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const readinessTimeout = 2 * time.Second

type healthRes struct {
	Status string `json:"status"`
} //@name HealthResponse

// Healthz is a liveness probe. It only reports that the HTTP process is up.
func (h *DefaultHandler) Healthz(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, healthRes{Status: "ok"})
}

// Readyz is a readiness probe. It verifies that the database is reachable.
func (h *DefaultHandler) Readyz(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")

	if h.store == nil {
		h.logger.Error().Msg("readiness check failed: store unavailable")
		ctx.JSON(http.StatusServiceUnavailable, healthRes{Status: "unavailable"})
		return
	}

	pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), readinessTimeout)
	defer cancel()

	if err := h.store.Ping(pingCtx); err != nil {
		h.logger.Error().Err(err).Msg("readiness check failed")
		ctx.JSON(http.StatusServiceUnavailable, healthRes{Status: "unavailable"})
		return
	}

	ctx.JSON(http.StatusOK, healthRes{Status: "ok"})
}
