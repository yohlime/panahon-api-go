package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/gin-gonic/gin"
)

func getAuthKey(tokenMaker token.Maker, ctx *gin.Context) (payload *token.Payload, err error) {
	accessToken, err := ctx.Cookie(models.AccessTokenCookieName)
	if err != nil {
		authHeader := ctx.GetHeader(models.AuthHeaderKey)
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
		if authType != models.AuthTypeBearer {
			err = fmt.Errorf("unsupported authorization authorization type %s", authType)
			return
		}

		accessToken = fields[1]
	}

	payload, err = tokenMaker.VerifyToken(accessToken)
	return
}

func AuthMiddleware(tokenMaker token.Maker, permissive bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := getAuthKey(tokenMaker, ctx)
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
