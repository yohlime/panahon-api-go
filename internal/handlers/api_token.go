package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateAPIToken
//
//	@Summary	Create API token for current user
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		req	body	models.CreateAPITokenReq	true	"Create API token parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.APITokenWithSecret
//	@Router		/users/me/api-tokens [post]
func (h *DefaultHandler) CreateAPIToken(ctx *gin.Context) {
	var req models.CreateAPITokenReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)

	user, err := h.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	req.Permissions.Normalize()

	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{
			Time:  *req.ExpiresAt,
			Valid: true,
		}
	}

	// Generate token
	tok := util.NewAPIToken("")

	arg := db.CreateAPITokenParams{
		UserID:      user.ID,
		Name:        req.Name,
		TokenHash:   tok.GetHash(),
		TokenPrefix: tok.GetPrefix(),
		Permissions: permissionsJSON,
		ExpiresAt:   expiresAt,
	}

	dbToken, err := h.store.CreateAPIToken(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	apiToken := models.NewAPIToken(dbToken)
	res := models.APITokenWithSecret{
		APIToken: apiToken,
		Token:    tok.GetValue(),
	}

	ctx.JSON(http.StatusCreated, res)
}

// ListAPITokens
//
//	@Summary	List API tokens for current user
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	models.APIToken
//	@Router		/users/me/api-tokens [get]
func (h *DefaultHandler) ListAPITokens(ctx *gin.Context) {
	authPayload := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)

	user, err := h.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	dbTokens, err := h.store.ListAPITokensByUser(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	tokens := make([]models.APIToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = models.NewAPIToken(dbToken)
	}

	ctx.JSON(http.StatusOK, tokens)
}

type deleteAPITokenUri struct {
	ID int64 `uri:"token_id" binding:"required,min=1"`
}

// DeleteAPIToken
//
//	@Summary	Delete API token for current user
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		token_id	path	int	true	"Token ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/users/me/api-tokens/{token_id} [delete]
func (h *DefaultHandler) DeleteAPIToken(ctx *gin.Context) {
	var uri deleteAPITokenUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)

	user, err := h.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify the token belongs to the current user
	dbToken, err := h.store.GetAPIToken(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("token not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if dbToken.UserID != user.ID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("cannot delete another user's token")))
		return
	}

	if err := h.store.DeleteAPIToken(ctx, uri.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// CreateUserAPIToken
//
//	@Summary	Create API token for any user (Admin only)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int							true	"User ID"
//	@Param		req	body	models.CreateAPITokenReq	true	"Create API token parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.APITokenWithSecret
//	@Router		/users/{id}/api-tokens [post]
func (h *DefaultHandler) CreateUserAPIToken(ctx *gin.Context) {
	var uri getUserReq
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.CreateAPITokenReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Verify user exists
	_, err := h.store.GetUser(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Set default permissions if not provided
	req.Permissions.Normalize()

	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{
			Time:  *req.ExpiresAt,
			Valid: true,
		}
	}

	// Generate token
	tok := util.NewAPIToken("")

	arg := db.CreateAPITokenParams{
		UserID:      uri.ID,
		Name:        req.Name,
		TokenHash:   tok.GetHash(),
		TokenPrefix: tok.GetPrefix(),
		Permissions: permissionsJSON,
		ExpiresAt:   expiresAt,
	}

	dbToken, err := h.store.CreateAPIToken(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	apiToken := models.NewAPIToken(dbToken)
	res := models.APITokenWithSecret{
		APIToken: apiToken,
		Token:    tok.GetValue(),
	}

	ctx.JSON(http.StatusCreated, res)
}

// ListUserAPITokens
//
//	@Summary	List API tokens for any user (Admin only)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{array}	models.APIToken
//	@Router		/users/{id}/api-tokens [get]
func (h *DefaultHandler) ListUserAPITokens(ctx *gin.Context) {
	var uri getUserReq
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	dbTokens, err := h.store.ListAPITokensByUser(ctx, uri.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	tokens := make([]models.APIToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = models.NewAPIToken(dbToken)
	}

	ctx.JSON(http.StatusOK, tokens)
}

type deleteUserAPITokenUri struct {
	UserID  int64 `uri:"id" binding:"required,min=1"`
	TokenID int64 `uri:"token_id" binding:"required,min:1"`
}

// DeleteUserAPIToken
//
//	@Summary	Delete API token for any user (Admin only)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id			path	int	true	"User ID"
//	@Param		token_id	path	int	true	"Token ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/users/{id}/api-tokens/{token_id} [delete]
func (h *DefaultHandler) DeleteUserAPIToken(ctx *gin.Context) {
	var uri deleteUserAPITokenUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Verify the token exists and belongs to the specified user
	dbToken, err := h.store.GetAPIToken(ctx, uri.TokenID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("token not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if dbToken.UserID != uri.UserID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("token does not belong to specified user")))
		return
	}

	if err := h.store.DeleteAPIToken(ctx, uri.TokenID); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
