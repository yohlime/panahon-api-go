package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
} //@name RenewAccessTokenParams

// RenewAccessToken
//
//	@Summary	Renew access token
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	renewAccessTokenRequest	true	"Renew access token parameters"
//	@Success	204
//	@Router		/tokens/renew [post]
func (h *DefaultHandler) RenewAccessToken(ctx *gin.Context) {
	refreshToken, err := ctx.Cookie(models.RefreshTokenCookieName)
	if err != nil {
		var req renewAccessTokenRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		refreshToken = req.RefreshToken
	}

	refreshPayload, err := h.tokenMaker.VerifyToken(refreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	session, err := h.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if session.IsBlocked {
		err := fmt.Errorf("blocked session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	user, err := h.store.GetUser(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if user.Username != refreshPayload.User.Username {
		err := fmt.Errorf("incorrect session user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.RefreshToken != refreshToken {
		err := fmt.Errorf("mismatched session token")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if time.Now().After(session.ExpiresAt.Time) {
		err := fmt.Errorf("expired session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := h.tokenMaker.CreateToken(refreshPayload.User, h.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	cookieIsSecure := h.config.Environment == "production"
	ctx.SetCookie(models.AccessTokenCookieName, accessToken, int(time.Until(accessPayload.ExpiresAt).Seconds()), h.config.CookiePath, h.config.CookieDomain, cookieIsSecure, true)

	ctx.JSON(http.StatusNoContent, nil)
}
