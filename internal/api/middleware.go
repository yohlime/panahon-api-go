package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/gin-gonic/gin"
)

const (
	authHeaderKey  = "authorization"
	authTypeBearer = "bearer"
	authPayloadKey = "authorization_payload"
)

func getAuthKey(tokenMaker token.Maker, ctx *gin.Context) (payload *token.Payload, err error) {
	authHeader := ctx.GetHeader(authHeaderKey)
	if len(authHeader) == 0 {
		err = errors.New("authorization header is not provided")
		return
	}

	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		err = errors.New("invalid authorization header format")
		return
	}

	authType := strings.ToLower(fields[0])
	if authType != authTypeBearer {
		err = fmt.Errorf("unsupported authorization authorization type %s", authType)
		return
	}

	accessToken := fields[1]

	payload, err = tokenMaker.VerifyToken(accessToken)
	return
}

func authMiddleware(tokenMaker token.Maker, permissive bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := getAuthKey(tokenMaker, ctx)
		if !permissive && err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authPayloadKey, payload)
		ctx.Next()
	}
}

func roleMiddleware(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authPayloadKey).(*token.Payload)

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

func adminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authPayloadKey).(*token.Payload)

		isAdmin := false
		for _, roleName := range authPayload.User.Roles {
			if isAdmin = isAdminRole(roleName); isAdmin {
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
