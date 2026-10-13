package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/gin-gonic/gin"
)

// setAuthCookie sets an httpOnly, SameSite=Lax auth cookie. The Secure flag is
// only enabled in production so local (http/https mixed) dev keeps working.
func (h *DefaultHandler) setAuthCookie(ctx *gin.Context, name, value string, expiresAt time.Time) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	secure := h.config.Environment == "production"
	ctx.SetCookie(name, value, int(time.Until(expiresAt).Seconds()), h.config.CookiePath, h.config.CookieDomain, secure, true)
}

func (h *DefaultHandler) clearAuthCookie(ctx *gin.Context, name string) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	secure := h.config.Environment == "production"
	ctx.SetCookie(name, "", -1, h.config.CookiePath, h.config.CookieDomain, secure, true)
}

// isAdminSession reports whether the request carries an authenticated
// admin/superadmin payload (cookie or Bearer).
func isAdminSession(ctx *gin.Context) bool {
	key, exists := ctx.Get(models.AuthPayloadKey)
	if !exists || key == nil {
		return false
	}
	payload, ok := key.(*token.Payload)
	if !ok || payload == nil {
		return false
	}
	for _, role := range payload.User.Roles {
		if models.IsAdminRole(role) {
			return true
		}
	}
	return false
}

// getAPIToken returns the validated API token attached to the request, if any.
func getAPIToken(ctx *gin.Context) (*db.APIToken, bool) {
	key, exists := ctx.Get(models.APITokenAuthKey)
	if !exists || key == nil {
		return nil, false
	}
	dbToken, ok := key.(*db.APIToken)
	if !ok || dbToken == nil {
		return nil, false
	}
	return dbToken, true
}

// getAPITokenPermissions decodes the permissions of the request's API token.
// The bool is false when there is no API token (e.g. an admin session).
func getAPITokenPermissions(ctx *gin.Context) (models.APITokenPermissions, bool) {
	dbToken, ok := getAPIToken(ctx)
	if !ok {
		return models.APITokenPermissions{}, false
	}

	var perm models.APITokenPermissions
	if err := json.Unmarshal(dbToken.Permissions, &perm); err != nil {
		return models.APITokenPermissions{}, true
	}
	return perm, true
}

// resolveVariables maps a token's variables list to the effective filter
// (nil = all variables).
func resolveVariables(vars []string) []string {
	if len(vars) == 0 {
		return models.DefaultAPITokenVariables
	}
	for _, v := range vars {
		if strings.EqualFold(v, models.APITokenAllVariables) {
			return nil
		}
	}
	return vars
}

// intersectInt64 returns the elements of a that are also present in b.
func intersectInt64(a, b []int64) []int64 {
	set := make(map[int64]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}

	out := make([]int64, 0, len(a))
	for _, v := range a {
		if _, ok := set[v]; ok {
			out = append(out, v)
		}
	}
	return out
}
