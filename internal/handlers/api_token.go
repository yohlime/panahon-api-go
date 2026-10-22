package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateAPIToken
//
//	@Summary	Create an API token for a user (self or admin)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int							true	"User ID"
//	@Param		req	body	models.CreateAPITokenReq	true	"Create API token parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.APITokenWithSecret
//	@Failure	400	"token scope has no stations"
//	@Failure	403	"owner has no station access"
//	@Router		/users/{id}/api-tokens [post]
func (h *DefaultHandler) CreateAPIToken(ctx *gin.Context) {
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

	if _, err := h.store.GetUser(ctx, uri.ID); err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Admin-owned tokens are not capped by the owner's entitlement: default to
	// everything when omitted, and honor an explicit (possibly narrower) scope.
	// Non-admin tokens default to, and are clamped to, the owner's entitlement.
	admin := h.isAdminUser(ctx, uri.ID)

	var access models.UserAccess
	if !admin {
		access = h.getUserAccess(ctx, uri.ID)
		// nil stations means "all"; an empty list means no stations were granted.
		if access.StationIDs != nil && len(access.StationIDs) == 0 {
			ctx.JSON(http.StatusForbidden, errorResponse(errors.New("no station access; request access first")))
			return
		}
	}

	var perms models.APITokenPermissions
	if req.Permissions == nil {
		if admin {
			perms = models.APITokenPermissions{MaxDaysHistory: -1}
		} else {
			perms = access.Permissions()
		}
	} else {
		requested := *req.Permissions
		requested.Normalize()
		if admin {
			perms = requested
		} else {
			perms = models.IntersectPermissions(requested, access)
		}
	}

	if !admin && perms.StationIDs != nil && len(perms.StationIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("token must grant at least one station")))
		return
	}

	permissionsJSON, err := json.Marshal(perms)
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

	res := models.APITokenWithSecret{
		APIToken: models.NewAPIToken(dbToken),
		Token:    tok.GetValue(),
	}

	ctx.JSON(http.StatusCreated, res)
}

// ListAPITokens
//
//	@Summary	List a user's API tokens (self or admin)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{array}	models.APIToken
//	@Router		/users/{id}/api-tokens [get]
func (h *DefaultHandler) ListAPITokens(ctx *gin.Context) {
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

type deleteAPITokenUri struct {
	ID      int64 `uri:"id" binding:"required,min=1"`
	TokenID int64 `uri:"token_id" binding:"required,min=1"`
}

// DeleteAPIToken
//
//	@Summary	Delete a user's API token (self or admin)
//	@Tags		api-tokens
//	@Accept		json
//	@Produce	json
//	@Param		id			path	int	true	"User ID"
//	@Param		token_id	path	int	true	"Token ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/users/{id}/api-tokens/{token_id} [delete]
func (h *DefaultHandler) DeleteAPIToken(ctx *gin.Context) {
	var uri deleteAPITokenUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	dbToken, err := h.store.GetAPIToken(ctx, uri.TokenID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("token not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if dbToken.UserID != uri.ID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("token does not belong to the user")))
		return
	}

	if err := h.store.DeleteAPIToken(ctx, uri.TokenID); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
