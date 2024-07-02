package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Zerologger(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		event := logger.Info()
		switch {
		case statusCode >= 500:
			event = logger.Error()
		case statusCode >= 400:
			event = logger.Warn()
		case statusCode >= 300:
			event = logger.Info()
		}

		event.Str("ip", clientIP).
			Str("method", method).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("path", path).
			Msg("[GIN]")
	}
}
