package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---- user_access cache -----------------------------------------------------

type accessCacheEntry struct {
	access  models.UserAccess
	expires time.Time
}

type accessCache struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[int64]accessCacheEntry
}

func newAccessCache(ttl time.Duration) *accessCache {
	return &accessCache{ttl: ttl, m: make(map[int64]accessCacheEntry)}
}

func (c *accessCache) get(userID int64) (models.UserAccess, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.m[userID]
	if !ok || time.Now().After(e.expires) {
		return models.UserAccess{}, false
	}
	return e.access, true
}

func (c *accessCache) set(a models.UserAccess) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[a.UserID] = accessCacheEntry{access: a, expires: time.Now().Add(c.ttl)}
}

func (c *accessCache) del(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, userID)
}

var userAccessCache = newAccessCache(30 * time.Second)

// ---- effective scope -------------------------------------------------------

// getUserAccess returns a user's entitlement, using the default when none exists.
func (h *DefaultHandler) getUserAccess(ctx context.Context, userID int64) models.UserAccess {
	if a, ok := userAccessCache.get(userID); ok {
		return a
	}

	access := models.DefaultUserAccess(userID)
	row, err := h.store.GetUserAccess(ctx, userID)
	switch {
	case err == nil:
		access = models.NewUserAccess(row)
	case errors.Is(err, db.ErrRecordNotFound):
		// keep defaults
	default:
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("cannot load user access")
	}

	userAccessCache.set(access)
	return access
}

func (h *DefaultHandler) setUserAccess(ctx context.Context, userID int64, access models.UserAccess) error {
	stationIDs, variables := models.MarshalUserAccess(access)
	row, err := h.store.UpsertUserAccess(ctx, db.UpsertUserAccessParams{
		UserID:         userID,
		StationIds:     stationIDs,
		Variables:      variables,
		MaxDaysHistory: int32(access.MaxDaysHistory),
	})
	if err != nil {
		return err
	}

	userAccessCache.set(models.NewUserAccess(row))
	return nil
}

// effectivePermissions returns the scope a request may read. restricted is
// false only for admin sessions (unrestricted). API-token requests are capped
// by the token permissions; other authenticated sessions are scoped by the
// user's entitlement (privileges are tied to the user, no API key needed).
func (h *DefaultHandler) effectivePermissions(ctx *gin.Context) (models.APITokenPermissions, bool) {
	if dbToken, ok := getAPIToken(ctx); ok {
		var perm models.APITokenPermissions
		if err := json.Unmarshal(dbToken.Permissions, &perm); err != nil {
			perm = models.APITokenPermissions{}
		}
		return models.IntersectPermissions(perm, h.getUserAccess(ctx, dbToken.UserID)), true
	}

	if isAdminSession(ctx) {
		return models.APITokenPermissions{}, false
	}

	if payload, ok := authPayload(ctx); ok {
		if user, err := h.store.GetUserByUsername(ctx, payload.User.Username); err == nil {
			return h.getUserAccess(ctx, user.ID).Permissions(), true
		}
	}

	// Unauthenticated: deny-by-default.
	return models.APITokenPermissions{
		StationIDs:     []int64{},
		Variables:      models.DefaultAPITokenVariables,
		MaxDaysHistory: models.DefaultMaxDaysHistory,
	}, true
}

func (h *DefaultHandler) resolveAccessibleVariables(ctx *gin.Context) []string {
	perm, restricted := h.effectivePermissions(ctx)
	if !restricted {
		return nil
	}
	return resolveVariables(perm.Variables)
}

func (h *DefaultHandler) effectiveHistoryStart(ctx *gin.Context) (time.Time, bool) {
	perm, restricted := h.effectivePermissions(ctx)
	if !restricted || perm.MaxDaysHistory < 0 {
		return time.Time{}, false
	}
	return time.Now().AddDate(0, 0, -perm.MaxDaysHistory), true
}

func (h *DefaultHandler) stationAllowed(ctx *gin.Context, stationID int64) bool {
	perm, restricted := h.effectivePermissions(ctx)
	if !restricted || perm.StationIDs == nil {
		return true
	}
	for _, id := range perm.StationIDs {
		if id == stationID {
			return true
		}
	}
	return false
}

// stationFilter returns the token's effective station cap (nil = all).
func (h *DefaultHandler) stationFilter(ctx *gin.Context) ([]int64, bool) {
	perm, restricted := h.effectivePermissions(ctx)
	if !restricted {
		return nil, false
	}
	return perm.StationIDs, true
}

// ---- helpers ---------------------------------------------------------------

func (h *DefaultHandler) authUser(ctx *gin.Context) (db.User, bool) {
	authPayload, ok := ctx.MustGet(models.AuthPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return db.User{}, false
	}

	user, err := h.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return db.User{}, false
	}
	return user, true
}

func marshalSlice[T any](v []T) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func toInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

// ---- user endpoints --------------------------------------------------------

// GetMyAccess
//
//	@Summary	Get my access
//	@Tags		access
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	models.UserAccess
//	@Router		/users/me/access [get]
func (h *DefaultHandler) GetMyAccess(ctx *gin.Context) {
	user, ok := h.authUser(ctx)
	if !ok {
		return
	}
	ctx.JSON(http.StatusOK, h.getUserAccess(ctx, user.ID))
}

// CreateMyAccessRequest
//
//	@Summary	Request additional access
//	@Tags		access
//	@Accept		json
//	@Produce	json
//	@Param		req	body	models.CreateAccessRequestReq	true	"Access request parameters"
//	@Security	BearerAuth
//	@Success	201	{object}	models.AccessRequest
//	@Router		/users/me/access-requests [post]
func (h *DefaultHandler) CreateMyAccessRequest(ctx *gin.Context) {
	user, ok := h.authUser(ctx)
	if !ok {
		return
	}

	var req models.CreateAccessRequestReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	row, err := h.store.CreateAccessRequest(ctx, db.CreateAccessRequestParams{
		UserID:         user.ID,
		StationIds:     marshalSlice(req.StationIDs),
		AllStations:    req.AllStations,
		Variables:      marshalSlice(req.Variables),
		AllVariables:   req.AllVariables,
		MaxDaysHistory: toInt4(req.MaxDaysHistory),
		Reason:         util.ToPgText(req.Reason),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, models.NewAccessRequest(row))
}

// ListMyAccessRequests
//
//	@Summary	List my access requests
//	@Tags		access
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{array}	models.AccessRequest
//	@Router		/users/me/access-requests [get]
func (h *DefaultHandler) ListMyAccessRequests(ctx *gin.Context) {
	user, ok := h.authUser(ctx)
	if !ok {
		return
	}

	rows, err := h.store.ListAccessRequestsByUser(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	items := make([]models.AccessRequest, len(rows))
	for i, row := range rows {
		items[i] = models.NewAccessRequest(row)
	}

	ctx.JSON(http.StatusOK, items)
}

type accessRequestUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// CancelMyAccessRequest
//
//	@Summary	Cancel a pending access request
//	@Tags		access
//	@Produce	json
//	@Param		id	path	int	true	"Request ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/users/me/access-requests/{id} [delete]
func (h *DefaultHandler) CancelMyAccessRequest(ctx *gin.Context) {
	user, ok := h.authUser(ctx)
	if !ok {
		return
	}

	var uri accessRequestUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	row, err := h.store.GetAccessRequest(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("request not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if row.UserID != user.ID || row.Status != models.AccessRequestPending {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("cannot cancel this request")))
		return
	}

	if err := h.store.DeleteAccessRequest(ctx, uri.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// ---- admin endpoints -------------------------------------------------------

type getUserAccessUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GetUserAccess
//
//	@Summary	Get a user's access entitlement
//	@Tags		access
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{object}	models.UserAccess
//	@Router		/users/{id}/access [get]
func (h *DefaultHandler) GetUserAccess(ctx *gin.Context) {
	var uri getUserAccessUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, h.getUserAccess(ctx, uri.ID))
}

// UpdateUserAccess
//
//	@Summary	Set a user's access entitlement
//	@Tags		access
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int							true	"User ID"
//	@Param		req	body	models.UpdateUserAccessReq	true	"Entitlement"
//	@Security	BearerAuth
//	@Success	200	{object}	models.UserAccess
//	@Router		/users/{id}/access [put]
func (h *DefaultHandler) UpdateUserAccess(ctx *gin.Context) {
	var uri getUserAccessUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.UpdateUserAccessReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	access := models.UserAccess{
		UserID:         uri.ID,
		Variables:      req.Variables,
		MaxDaysHistory: req.MaxDaysHistory,
	}
	if req.AllStations {
		access.StationIDs = nil
	} else {
		access.StationIDs = req.StationIDs
		if access.StationIDs == nil {
			access.StationIDs = []int64{}
		}
	}
	if req.AllVariables {
		access.Variables = nil
	}

	if err := h.setUserAccess(ctx, uri.ID, access); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, h.getUserAccess(ctx, uri.ID))
}

type listAccessRequestsReq struct {
	Page    int32  `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int32  `form:"per_page,default=10" binding:"omitempty,min=1,max=100"`
	Status  string `form:"status" binding:"omitempty,oneof=pending approved denied"`
} //@name ListAccessRequestsParams

type paginatedAccessRequests = util.PaginatedList[models.AccessRequest] //@name PaginatedAccessRequests

// ListAccessRequests
//
//	@Summary	List access requests
//	@Tags		access
//	@Produce	json
//	@Param		req	query	listAccessRequestsReq	false	"List access requests parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	paginatedAccessRequests
//	@Router		/access-requests [get]
func (h *DefaultHandler) ListAccessRequests(ctx *gin.Context) {
	var req listAccessRequestsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	var (
		rows  []db.AccessRequest
		count int64
		err   error
	)
	if req.Status != "" {
		rows, err = h.store.ListAccessRequestsByStatus(ctx, db.ListAccessRequestsByStatusParams{
			Status: req.Status,
			Limit:  req.PerPage,
			Offset: offset,
		})
		if err == nil {
			count, err = h.store.CountAccessRequestsByStatus(ctx, req.Status)
		}
	} else {
		rows, err = h.store.ListAccessRequests(ctx, db.ListAccessRequestsParams{
			Limit:  req.PerPage,
			Offset: offset,
		})
		if err == nil {
			count, err = h.store.CountAccessRequests(ctx)
		}
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	items := make([]models.AccessRequest, len(rows))
	for i, row := range rows {
		items[i] = models.NewAccessRequest(row)
	}

	ctx.JSON(http.StatusOK, util.NewPaginatedList(req.Page, req.PerPage, int32(count), items))
}

type reviewAccessRequestUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// ApproveAccessRequest
//
//	@Summary	Approve an access request (optionally a subset)
//	@Tags		access
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int								true	"Request ID"
//	@Param		req	body	models.ReviewAccessRequestReq	false	"Optional partial approval / note"
//	@Security	BearerAuth
//	@Success	200	{object}	models.UserAccess
//	@Router		/access-requests/{id}/approve [post]
func (h *DefaultHandler) ApproveAccessRequest(ctx *gin.Context) {
	reviewer, ok := h.authUser(ctx)
	if !ok {
		return
	}

	var uri reviewAccessRequestUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.ReviewAccessRequestReq
	_ = ctx.ShouldBindJSON(&req)

	row, err := h.store.GetAccessRequest(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("request not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if row.Status != models.AccessRequestPending {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("request already reviewed")))
		return
	}
	if row.UserID == reviewer.ID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("cannot approve your own request")))
		return
	}

	current := h.getUserAccess(ctx, row.UserID)
	var merged models.UserAccess
	if hasReviewOverride(req) {
		merged = models.ApplyReview(current, req)
	} else {
		merged = models.MergeRequest(current, models.NewAccessRequest(row).ToCreateReq())
	}

	if err := h.setUserAccess(ctx, row.UserID, merged); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if _, err := h.store.UpdateAccessRequestStatus(ctx, db.UpdateAccessRequestStatusParams{
		ID:         row.ID,
		Status:     models.AccessRequestApproved,
		ReviewedBy: pgtype.Int8{Int64: reviewer.ID, Valid: true},
		ReviewNote: util.ToPgText(req.Note),
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, h.getUserAccess(ctx, row.UserID))
}

// DenyAccessRequest
//
//	@Summary	Deny an access request
//	@Tags		access
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int								true	"Request ID"
//	@Param		req	body	models.ReviewAccessRequestReq	true	"Denial (note required)"
//	@Security	BearerAuth
//	@Success	200	{object}	models.AccessRequest
//	@Router		/access-requests/{id}/deny [post]
func (h *DefaultHandler) DenyAccessRequest(ctx *gin.Context) {
	reviewer, ok := h.authUser(ctx)
	if !ok {
		return
	}

	var uri reviewAccessRequestUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req models.ReviewAccessRequestReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.Note == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("a note is required to deny")))
		return
	}

	row, err := h.store.GetAccessRequest(ctx, uri.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("request not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if row.Status != models.AccessRequestPending {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("request already reviewed")))
		return
	}

	updated, err := h.store.UpdateAccessRequestStatus(ctx, db.UpdateAccessRequestStatusParams{
		ID:         row.ID,
		Status:     models.AccessRequestDenied,
		ReviewedBy: pgtype.Int8{Int64: reviewer.ID, Valid: true},
		ReviewNote: util.ToPgText(req.Note),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, models.NewAccessRequest(updated))
}

func hasReviewOverride(req models.ReviewAccessRequestReq) bool {
	return req.AllStations || req.AllVariables ||
		len(req.StationIDs) > 0 || len(req.Variables) > 0 || req.MaxDaysHistory != nil
}
