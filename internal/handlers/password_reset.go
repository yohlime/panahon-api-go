package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/session"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultPasswordResetTokenDuration = time.Hour

var errInvalidPasswordResetToken = errors.New("invalid or expired password reset link")

type passwordResetLinkRes struct {
	ResetURL string `json:"reset_url"`
} //@name PasswordResetLinkResponse

func (h *DefaultHandler) passwordResetTokenDuration() time.Duration {
	if h.config.PasswordResetTokenDuration > 0 {
		return h.config.PasswordResetTokenDuration
	}
	return defaultPasswordResetTokenDuration
}

func (h *DefaultHandler) passwordResetURL(token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(h.config.FrontendBaseURL), "/")
	if base == "" {
		if h.config.Environment == "production" {
			return "", errors.New("FRONTEND_BASE_URL is not configured")
		}
		base = "http://localhost:8000"
	}
	return base + "/reset-password?token=" + url.QueryEscape(token), nil
}

type createPasswordResetLinkUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// CreatePasswordResetLink
//
//	@Summary	Create a password reset link for an active user
//	@Tags		users
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{object}	passwordResetLinkRes
//	@Router		/users/{id}/password-reset-link [post]
func (h *DefaultHandler) CreatePasswordResetLink(ctx *gin.Context) {
	var uri createPasswordResetLinkUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	admin, ok := h.authUser(ctx)
	if !ok {
		return
	}

	user, err := h.store.GetUser(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if user.Status != models.UserStatusActive {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("user is not active")))
		return
	}

	rawToken, err := util.GenerateSecureToken(32)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if _, err := h.store.CreatePasswordResetTokenTx(ctx, db.CreatePasswordResetTokenTxParams{
		UserID:    user.ID,
		TokenHash: util.HashToken(rawToken),
		ExpiresAt: time.Now().Add(h.passwordResetTokenDuration()),
		CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}); err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.ID).Msg("cannot create password reset link")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot create password reset link")))
		return
	}

	resetURL, err := h.passwordResetURL(rawToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, passwordResetLinkRes{ResetURL: resetURL})
}

type validatePasswordResetReq struct {
	Token string `json:"token" binding:"required"`
} //@name ValidatePasswordResetParams

type passwordResetInfoRes struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
} //@name PasswordResetInfo

func (h *DefaultHandler) getPasswordResetUser(ctx *gin.Context, rawToken string) (db.UserToken, db.User, error) {
	if rawToken == "" {
		return db.UserToken{}, db.User{}, errInvalidPasswordResetToken
	}

	token, err := h.store.GetUserTokenByHash(ctx, db.GetUserTokenByHashParams{
		TokenHash: util.HashToken(rawToken),
		Purpose:   db.UserTokenPurposePasswordReset,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return db.UserToken{}, db.User{}, errInvalidPasswordResetToken
		}
		return db.UserToken{}, db.User{}, err
	}

	if token.UsedAt.Valid || !token.ExpiresAt.Valid || time.Now().After(token.ExpiresAt.Time) {
		return db.UserToken{}, db.User{}, errInvalidPasswordResetToken
	}

	user, err := h.store.GetUser(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return db.UserToken{}, db.User{}, errInvalidPasswordResetToken
		}
		return db.UserToken{}, db.User{}, err
	}

	if user.Status != models.UserStatusActive {
		return db.UserToken{}, db.User{}, errInvalidPasswordResetToken
	}

	return token, user, nil
}

// ValidatePasswordReset
//
//	@Summary	Validate a password reset link
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body		validatePasswordResetReq	true	"Password reset token"
//	@Success	200	{object}	passwordResetInfoRes
//	@Router		/users/password-reset/validate [post]
func (h *DefaultHandler) ValidatePasswordReset(ctx *gin.Context) {
	var req validatePasswordResetReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, user, err := h.getPasswordResetUser(ctx, req.Token)
	if err != nil {
		if errors.Is(err, errInvalidPasswordResetToken) {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		h.logger.Error().Err(err).Msg("cannot validate password reset link")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot validate password reset link")))
		return
	}

	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, passwordResetInfoRes{
		Username: user.Username,
		FullName: user.FullName,
		Email:    email,
	})
}

type acceptPasswordResetReq struct {
	Token           string `json:"token" binding:"required"`
	Password        string `json:"password" binding:"required,min=8,eqfield=ConfirmPassword"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8"`
} //@name AcceptPasswordResetParams

// AcceptPasswordReset
//
//	@Summary	Set a new password using a reset link
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	acceptPasswordResetReq	true	"New password"
//	@Success	204
//	@Router		/users/password-reset/accept [post]
func (h *DefaultHandler) AcceptPasswordReset(ctx *gin.Context) {
	var req acceptPasswordResetReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if _, _, err := h.getPasswordResetUser(ctx, req.Token); err != nil {
		if errors.Is(err, errInvalidPasswordResetToken) {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		h.logger.Error().Err(err).Msg("cannot validate password reset link")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot reset password")))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	user, err := h.store.ResetPasswordTx(ctx, db.ResetPasswordTxParams{
		TokenHash: util.HashToken(req.Token),
		Password:  hashedPassword,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidPasswordResetToken))
			return
		}
		h.logger.Error().Err(err).Msg("cannot reset password")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot reset password")))
		return
	}

	session.Clear()
	if err := h.startSession(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("cannot start session after password reset")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("password reset; please log in")))
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusNoContent, nil)
}
