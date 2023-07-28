package api

import (
	"errors"
	"net/http"
	"strings"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type roleResponse struct {
	Name        string             `json:"name"`
	Description util.NullString    `json:"description"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
} //@name RoleResponse

func newRoleResponse(role db.Role) roleResponse {
	return roleResponse{
		Name:        role.Name,
		Description: role.Description,
		UpdatedAt:   role.UpdatedAt,
		CreatedAt:   role.CreatedAt,
	}
}

type listRoleReq struct {
	Page  int32 `form:"page,default=1" binding:"omitempty,min=1"`
	Limit int32 `form:"limit,default=5" binding:"omitempty,min=1,max=30"`
} //@name ListRolesParams

// ListRoles godoc
//
//	@Summary	List roles
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		req	query	listRoleReq	false	"List roles parameters"
//	@Security	BearerAuth
//	@Success	200	{array}	roleResponse
//	@Router		/roles [get]
func (s *Server) ListRoles(ctx *gin.Context) {
	var req listRoleReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.Limit
	arg := db.ListRolesParams{
		Limit:  req.Limit,
		Offset: offset,
	}
	roles, err := s.store.ListRoles(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numRoles := len(roles)
	if numRoles <= 0 {
		ctx.JSON(http.StatusOK, nil)
	}

	rsp := make([]roleResponse, numRoles)
	for i, role := range roles {
		rsp[i] = newRoleResponse(role)
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getRoleReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GetRole godoc
//
//	@Summary	Get role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"Role ID"
//	@Security	BearerAuth
//	@Success	200	{object}	roleResponse
//	@Router		/roles/{id} [get]
func (s *Server) GetRole(ctx *gin.Context) {
	var req getRoleReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	role, err := s.store.GetRole(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("role not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newRoleResponse(role))
}

type createRoleReq struct {
	Name        string          `json:"name" binding:"required,alphanum"`
	Description util.NullString `json:"description" binding:"omitempty,alphanum"`
} //@name CreateRoleParams

// CreateRole godoc
//
//	@Summary	Create role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		req	body	createRoleReq	true	"Create role parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	roleResponse
//	@Router		/roles/{id} [post]
func (s *Server) CreateRole(ctx *gin.Context) {
	var req createRoleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateRoleParams{
		Name:        strings.ToUpper(req.Name),
		Description: req.Description,
	}

	role, err := s.store.CreateRole(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newRoleResponse(role))
}

type updateRoleUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateRoleReq struct {
	Name        util.NullString `json:"name" binding:"omitempty,alphanum"`
	Description util.NullString `json:"description" binding:"omitempty,alphanum"`
} //@name UpdateRoleParams

// UpdateRole godoc
//
//	@Summary	Update role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int				true	"Role ID"
//	@Param		req	body	updateRoleReq	true	"Update role parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	roleResponse
//	@Router		/roles/{id} [put]
func (s *Server) UpdateRole(ctx *gin.Context) {
	var uri updateRoleUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateRoleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if req.Name.Valid {
		req.Name.Text.String = strings.ToUpper(req.Name.Text.String)
	}

	arg := db.UpdateRoleParams{
		ID:          uri.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	role, err := s.store.UpdateRole(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("role not found")))
			return
		} else if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, newRoleResponse(role))
}

type deleteRoleReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// DeleteRole godoc
//
//	@Summary	Delete role
//	@Accept		json
//	@Tags		roles
//	@Produce	json
//	@Param		id	path	int	true	"Role ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/roles/{id} [delete]
func (s *Server) DeleteRole(ctx *gin.Context) {
	var req deleteRoleReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := s.store.DeleteRole(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
