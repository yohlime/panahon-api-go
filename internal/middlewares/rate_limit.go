package middlewares

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/ratelimit"
	"github.com/gin-gonic/gin"
)

// RateLimit limits requests per client IP within a fixed window. A non-positive
// limit disables the middleware.
func RateLimit(limiter *ratelimit.Limiter, name string, limit int, window time.Duration) gin.HandlerFunc {
	return rateLimit(limiter, name, limit, window, func(ctx *gin.Context) string {
		return ctx.ClientIP()
	}, nil)
}

// APIKeyRateLimit limits API-key requests per client IP before the key is
// validated, so invalid keys cannot be guessed without cost. Requests without
// an api_key query parameter are ignored.
func APIKeyRateLimit(limiter *ratelimit.Limiter, limit int, window time.Duration) gin.HandlerFunc {
	return rateLimit(limiter, "api_key", limit, window, func(ctx *gin.Context) string {
		return ctx.ClientIP()
	}, func(ctx *gin.Context) bool {
		return ctx.Query(models.APIKeyQueryParam) == ""
	})
}

func rateLimit(
	limiter *ratelimit.Limiter,
	name string,
	limit int,
	window time.Duration,
	key func(*gin.Context) string,
	skip func(*gin.Context) bool,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if limiter == nil || limit <= 0 || window <= 0 || (skip != nil && skip(ctx)) {
			ctx.Next()
			return
		}

		retryAfter, ok := limiter.Allow(name+":"+key(ctx), limit, window)
		if !ok {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			ctx.Header("Retry-After", strconv.Itoa(seconds))
			ctx.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				errorResponse(errors.New("too many requests; please try again later")),
			)
			return
		}

		ctx.Next()
	}
}
