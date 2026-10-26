package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultRegistrationTokenDuration = 72 * time.Hour

var errInvalidRegistrationToken = errors.New("invalid or expired registration link")

type registrationLinkRes struct {
	RegistrationURL string `json:"registration_url"`
} //@name RegistrationLinkResponse

func (h *DefaultHandler) registrationTokenDuration() time.Duration {
	if h.config.RegistrationTokenDuration > 0 {
		return h.config.RegistrationTokenDuration
	}
	return defaultRegistrationTokenDuration
}

func (h *DefaultHandler) registrationURL(token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(h.config.FrontendBaseURL), "/")
	if base == "" {
		if h.config.Environment == "production" {
			return "", errors.New("FRONTEND_BASE_URL is not configured")
		}
		base = "http://localhost:8000"
	}
	return base + "/complete-registration?token=" + url.QueryEscape(token), nil
}

func (h *DefaultHandler) createRegistrationLink(ctx *gin.Context, userID int64) (string, error) {
	rawToken, err := util.GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	if _, err := h.store.CreateRegistrationTokenTx(ctx, db.CreateRegistrationTokenTxParams{
		UserID:    userID,
		TokenHash: util.HashToken(rawToken),
		ExpiresAt: time.Now().Add(h.registrationTokenDuration()),
	}); err != nil {
		return "", err
	}

	return h.registrationURL(rawToken)
}

func (h *DefaultHandler) getRegistrationUser(ctx *gin.Context, rawToken string) (db.UserToken, db.User, error) {
	if rawToken == "" {
		return db.UserToken{}, db.User{}, errInvalidRegistrationToken
	}

	token, err := h.store.GetUserTokenByHash(ctx, db.GetUserTokenByHashParams{
		TokenHash: util.HashToken(rawToken),
		Purpose:   db.UserTokenPurposeRegistration,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return db.UserToken{}, db.User{}, errInvalidRegistrationToken
		}
		return db.UserToken{}, db.User{}, err
	}

	if token.UsedAt.Valid || !token.ExpiresAt.Valid || time.Now().After(token.ExpiresAt.Time) {
		return db.UserToken{}, db.User{}, errInvalidRegistrationToken
	}

	user, err := h.store.GetUser(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return db.UserToken{}, db.User{}, errInvalidRegistrationToken
		}
		return db.UserToken{}, db.User{}, err
	}

	if user.Status != models.UserStatusInvited {
		return db.UserToken{}, db.User{}, errInvalidRegistrationToken
	}

	return token, user, nil
}

type createRegistrationLinkUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// CreateRegistrationLink
//
//	@Summary	Create or regenerate a user's registration link
//	@Tags		users
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{object}	registrationLinkRes
//	@Router		/users/{id}/registration-link [post]
func (h *DefaultHandler) CreateRegistrationLink(ctx *gin.Context) {
	var uri createRegistrationLinkUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
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

	if user.Status != models.UserStatusInvited {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("user is not awaiting registration")))
		return
	}

	registrationURL, err := h.createRegistrationLink(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.ID).Msg("cannot create registration link")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot create registration link")))
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, registrationLinkRes{RegistrationURL: registrationURL})
}

type validateRegistrationReq struct {
	Token string `json:"token" binding:"required"`
} //@name ValidateRegistrationParams

type registrationInfoRes struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
} //@name RegistrationInfo

// ValidateCompleteRegistration
//
//	@Summary	Validate a registration link
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body		validateRegistrationReq	true	"Registration token"
//	@Success	200	{object}	registrationInfoRes
//	@Router		/users/complete-registration/validate [post]
func (h *DefaultHandler) ValidateCompleteRegistration(ctx *gin.Context) {
	var req validateRegistrationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, user, err := h.getRegistrationUser(ctx, req.Token)
	if err != nil {
		if errors.Is(err, errInvalidRegistrationToken) {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}
		h.logger.Error().Err(err).Msg("cannot validate registration link")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot validate registration link")))
		return
	}

	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, registrationInfoRes{
		Username: user.Username,
		FullName: user.FullName,
		Email:    email,
	})
}

type completeRegistrationReq struct {
	Token           string  `json:"token" binding:"required"`
	FullName        *string `json:"full_name" binding:"omitempty"`
	Email           *string `json:"email"`
	Password        string  `json:"password" binding:"required,min=6,eqfield=ConfirmPassword"`
	ConfirmPassword string  `json:"confirm_password" binding:"required,min=6"`
} //@name CompleteRegistrationParams

// CompleteRegistration
//
//	@Summary	Complete an invited user's registration
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	completeRegistrationReq	true	"Registration completion parameters"
//	@Success	204
//	@Router		/users/complete-registration/accept [post]
func (h *DefaultHandler) CompleteRegistration(ctx *gin.Context) {
	var req completeRegistrationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := validateOptionalEmail(req.Email); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var fullName pgtype.Text
	if req.FullName != nil {
		fullName = pgtype.Text{String: *req.FullName, Valid: true}
	}

	var email pgtype.Text
	if req.Email != nil {
		email = util.ToPgText(util.NormalizeEmail(*req.Email))
	}

	user, err := h.store.CompleteRegistrationTx(ctx, db.CompleteRegistrationTxParams{
		TokenHash: util.HashToken(req.Token),
		Password:  hashedPassword,
		FullName:  fullName,
		Email:     email,
		SetEmail:  req.Email != nil,
	})
	if err != nil {
		switch {
		case errors.Is(err, db.ErrRecordNotFound):
			ctx.JSON(http.StatusBadRequest, errorResponse(errInvalidRegistrationToken))
		case db.ErrorCode(err) == db.UniqueViolation:
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("email is already in use")))
		default:
			h.logger.Error().Err(err).Msg("cannot complete registration")
			ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot complete registration")))
		}
		return
	}

	if err := h.startSession(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("cannot start session after registration")
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("cannot start session")))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
