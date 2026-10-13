package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
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

func getAccessibleVariables(ctx *gin.Context) []string {
	key, exists := ctx.Get(models.APITokenAuthKey)
	if !exists {
		return nil
	}

	dbToken, ok := key.(*db.APIToken)
	if !ok || dbToken == nil {
		return nil
	}

	var perm models.APITokenPermissions
	err := json.Unmarshal(dbToken.Permissions, &perm)
	if err != nil {
		return nil
	}

	if perm.Variables == nil {
		return nil
	}

	return perm.Variables
}
