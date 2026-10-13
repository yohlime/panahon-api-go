package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
)

// isAuthenticated reports whether the request carries an authenticated session
// (cookie or Bearer) of any role.
func isAuthenticated(ctx *gin.Context) bool {
	key, exists := ctx.Get(models.AuthPayloadKey)
	if !exists || key == nil {
		return false
	}
	payload, ok := key.(*token.Payload)
	return ok && payload != nil
}

// RequireAuthOrAPIToken allows any authenticated session or a validated API
// token, else 401. Access scope is enforced by the handlers.
func RequireAuthOrAPIToken() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if isAuthenticated(ctx) {
			ctx.Next()
			return
		}

		if key, exists := ctx.Get(models.APITokenAuthKey); exists {
			if dbToken, ok := key.(*db.APIToken); ok && dbToken != nil {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(errors.New("authentication required")))
	}
}

func APITokenMiddleware(store db.Store, permissive bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		apiToken, err := getAPITokenKey(store, ctx)
		if !permissive && err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(models.APITokenAuthKey, apiToken)
		ctx.Next()
	}
}

func StationMiddleware(stnIDParamKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Any authenticated session may read; the handler enforces the user's
		// entitlement station scope (no API key needed for logged-in users).
		if isAuthenticated(ctx) {
			ctx.Next()
			return
		}

		key, exists := ctx.Get(models.APITokenAuthKey)
		if !exists {
			err := fmt.Errorf("api_key not found")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		dbToken, ok := key.(*db.APIToken)
		if !ok || dbToken == nil {
			err := fmt.Errorf("token not supported")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		if stnIDParamKey == "" {
			stnIDParamKey = "station_id"
		}
		idStr := ctx.Param(stnIDParamKey)
		stnID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		var perm models.APITokenPermissions
		err = json.Unmarshal(dbToken.Permissions, &perm)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		if perm.StationIDs == nil {
			ctx.Next()
			return
		}

		if !slices.Contains(perm.StationIDs, stnID) {
			err := fmt.Errorf("user cannot access this station")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		ctx.Next()
	}
}

func getAPITokenKey(store db.Store, ctx *gin.Context) (apiToken *db.APIToken, err error) {
	apiKey := ctx.Query(models.APIKeyQueryParam)

	if apiKey != "" {
		tok := util.NewAPIToken(apiKey)
		err = tok.VerifyToken()
		if err != nil {
			return
		}
		apiToken, err = validateAPIToken(ctx, store, *tok)
		return
	}

	err = errors.New("no supported authorization scheme provided")
	return
}

func validateAPIToken(ctx context.Context, store db.Store, tok util.APIToken) (*db.APIToken, error) {
	dbToken, err := store.GetAPITokenByHash(ctx, tok.GetHash())
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, err
	}

	// Check if token is expired
	if dbToken.ExpiresAt.Valid && time.Now().After(dbToken.ExpiresAt.Time) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Update last used time (async - don't block request)
	go func() {
		_ = store.UpdateAPITokenLastUsedAt(context.Background(), dbToken.ID)
	}()

	return &dbToken, nil
}
