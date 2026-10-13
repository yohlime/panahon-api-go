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

// getAPITokenPermissions decodes the permissions of the request's API token.
// The bool is false when there is no API token (e.g. an admin session).
func getAPITokenPermissions(ctx *gin.Context) (models.APITokenPermissions, bool) {
	key, exists := ctx.Get(models.APITokenAuthKey)
	if !exists || key == nil {
		return models.APITokenPermissions{}, false
	}
	dbToken, ok := key.(*db.APIToken)
	if !ok || dbToken == nil {
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

// resolveAccessibleVariables returns the met variables the caller may read
// (nil = all). Admin sessions and missing tokens are unrestricted.
func resolveAccessibleVariables(ctx *gin.Context) []string {
	if isAdminSession(ctx) {
		return nil
	}
	perm, ok := getAPITokenPermissions(ctx)
	if !ok {
		return nil
	}
	return resolveVariables(perm.Variables)
}

// effectiveHistoryStart returns the oldest timestamp a scoped caller may query.
// The bool is false when the caller is unrestricted (admin) or unauthenticated.
func effectiveHistoryStart(ctx *gin.Context) (time.Time, bool) {
	if isAdminSession(ctx) {
		return time.Time{}, false
	}
	perm, ok := getAPITokenPermissions(ctx)
	if !ok {
		return time.Time{}, false
	}

	days := perm.MaxDaysHistory
	if days <= 0 || days > models.DefaultMaxDaysHistory {
		days = models.DefaultMaxDaysHistory
	}
	return time.Now().AddDate(0, 0, -days), true
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
