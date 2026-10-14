package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/session"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getAuthKey(tokenMaker token.Maker, ctx *gin.Context) (payload *token.Payload, err error) {
	// Try cookie first
	accessToken, err := ctx.Cookie(models.AccessTokenCookieName)
	if err == nil && accessToken != "" {
		payload, err = tokenMaker.VerifyToken(accessToken)
		return
	}

	// Try Authorization header (Bearer token)
	authHeader := ctx.GetHeader(models.AuthHeaderKey)
	if authHeader != "" {
		fields := strings.Fields(authHeader)
		if len(fields) >= 2 {
			authType := strings.ToLower(fields[0])
			if authType == "bearer" {
				payload, err = tokenMaker.VerifyToken(fields[1])
				return
			}
		}
	}

	err = errors.New("no supported authorization scheme provided")
	return
}

func AuthMiddleware(tokenMaker token.Maker, store db.Store, permissive bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := getAuthKey(tokenMaker, ctx)

		// A verified token bound to a session only authenticates while that
		// session is alive (exists, not blocked, not expired). This is what
		// makes logout revoke access tokens instead of waiting for expiry.
		if err == nil && payload != nil && payload.SessionID != uuid.Nil {
			if !session.Live(ctx, store, payload.SessionID) {
				payload = nil
				err = errors.New("session is no longer valid")
			}
		}

		if !permissive && err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(models.AuthPayloadKey, payload)
		ctx.Next()
	}
}

func RoleMiddleware(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)

		hasRole := false
		for _, roleName := range authPayload.User.Roles {
			if role == roleName {
				hasRole = true
				break
			}
		}

		if !hasRole {
			err := fmt.Errorf("user has no role %s", role)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
		}

		ctx.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)

		isAdmin := false
		for _, roleName := range authPayload.User.Roles {
			if isAdmin = models.IsAdminRole(roleName); isAdmin {
				break
			}
		}

		if !isAdmin {
			err := fmt.Errorf("user has no admin access")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
		}

		ctx.Next()
	}
}
