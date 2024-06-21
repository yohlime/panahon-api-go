package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
)

type adminRoleType string

const (
	adminRole      adminRoleType = "ADMIN"
	superAdminRole adminRoleType = "SUPERADMIN"
)

func (rt adminRoleType) IsValid() error {
	switch rt {
	case adminRole, superAdminRole:
		return nil
	}
	return fmt.Errorf("invalid admin_role type")
}

func isAdminRole(role string) bool {
	rt := adminRoleType(role)
	if err := rt.IsValid(); err != nil {
		return false
	}
	return true
}

type createRoleReq struct {
	Name        string `json:"name" binding:"required,alphanum"`
	Description string `json:"description" binding:"alphanumspace"`
} //@name CreateRoleParams

// CreateRole
//
//	@Summary	Create role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		req	body	createRoleReq	true	"Create role parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.Role
//	@Router		/roles/{id} [post]
func (s *Server) CreateRole(ctx *gin.Context) {
	var req createRoleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateRoleParams{
		Name:        strings.ToUpper(req.Name),
		Description: util.ToPgText(req.Description),
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

	ctx.JSON(http.StatusOK, models.NewRole(role))
}

type listRolesReq struct {
	Page    int32 `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int32 `form:"per_page,default=5" binding:"omitempty,min=1,max=30"`
} //@name ListRolesParams

type paginatedRoles = util.PaginatedList[models.Role] //@name PaginatedRoles

// ListRoles
//
//	@Summary	List roles
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		req	query	listRolesReq	false	"List roles parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	paginatedRoles
//	@Router		/roles [get]
func (s *Server) ListRoles(ctx *gin.Context) {
	var req listRolesReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListRolesParams{
		Limit:  req.PerPage,
		Offset: offset,
	}
	roles, err := s.store.ListRoles(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numRoles := len(roles)
	items := make([]models.Role, numRoles)
	for i, role := range roles {
		items[i] = models.NewRole(role)
	}

	count, err := s.store.CountRoles(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList(req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
}

type getRoleReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GetRole
//
//	@Summary	Get role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"Role ID"
//	@Security	BearerAuth
//	@Success	200	{object}	models.Role
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

	ctx.JSON(http.StatusOK, models.NewRole(role))
}

type updateRoleUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateRoleReq struct {
	Name        string `json:"name" binding:"omitempty,alphanum"`
	Description string `json:"description" binding:"omitempty,alphanumspace"`
} //@name UpdateRoleParams

// UpdateRole
//
//	@Summary	Update role
//	@Tags		roles
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int				true	"Role ID"
//	@Param		req	body	updateRoleReq	true	"Update role parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.Role
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

	if len(req.Name) > 0 {
		req.Name = strings.ToUpper(req.Name)
	}

	arg := db.UpdateRoleParams{
		ID:          uri.ID,
		Name:        util.ToPgText(req.Name),
		Description: util.ToPgText(req.Description),
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

	ctx.JSON(http.StatusOK, models.NewRole(role))
}

type deleteRoleReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// DeleteRole
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
