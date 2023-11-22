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
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

func getAuthKey(tokenMaker token.Maker, ctx *gin.Context) (payload *token.Payload, err error) {
	authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
	if len(authorizationHeader) == 0 {
		err = errors.New("authorization header is not provided")
		return
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		err = errors.New("invalid authorization header format")
		return
	}

	authorizationType := strings.ToLower(fields[0])
	if authorizationType != authorizationTypeBearer {
		err = fmt.Errorf("unsupported authorization authorization type %s", authorizationType)
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

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

func roleMiddleware(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

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
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

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
