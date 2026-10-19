package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/session"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserReq struct {
	Username string   `json:"username" binding:"required,alphanum"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name" binding:"required,fullname"`
	Email    string   `json:"email" binding:"required,email"`
	Roles    []string `json:"roles"`
} //@name CreateUserParams

// CreateUser
//
//	@Summary	Create user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	createUserReq	true	"Create user parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Router		/users/{id} [post]
func (h *DefaultHandler) CreateUser(ctx *gin.Context) {
	var req createUserReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if !isSuperAdminSession(ctx) && containsSuperAdminRole(req.Roles) {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("only a superadmin can grant the SUPERADMIN role")))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username: req.Username,
		Password: hashedPassword,
		FullName: req.FullName,
		Email:    req.Email,
		Status:   models.UserStatusActive,
	}

	user, err := h.store.CreateUser(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(errors.New("username or email is already taken")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var roleNames []string
	if len(req.Roles) > 0 {
		var createUserRolesArgs []db.UserRolesParams
		for _, roleName := range req.Roles {
			createUserRolesArgs = append(createUserRolesArgs, db.UserRolesParams{
				RoleName: strings.ToUpper(roleName),
				Username: arg.Username,
			})
		}
		userRoles, _ := h.store.BulkCreateUserRoles(ctx, createUserRolesArgs)

		for _, userRole := range userRoles {
			roleNames = append(roleNames, userRole.RoleName)
		}
	}

	ctx.JSON(http.StatusOK, models.NewUser(user, roleNames))
}

type listUsersReq struct {
	Page    int32  `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int32  `form:"per_page,default=5" binding:"omitempty,min=1,max=30"`
	Status  string `form:"status" binding:"omitempty,oneof=pending active rejected"`
} //@name ListUsersParams

type paginatedUsers = util.PaginatedList[models.User] //@name PaginatedUsers

// ListUsers
//
//	@Summary	List users
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	query	listUsersReq	false	"List users parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	paginatedUsers
//	@Router		/users [get]
func (h *DefaultHandler) ListUsers(ctx *gin.Context) {
	var req listUsersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListUsersParams{
		Limit:  req.PerPage,
		Offset: offset,
		Status: util.ToPgText(req.Status),
	}
	users, err := h.store.ListUsers(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Roles are fetched in one query for the whole page.
	userIDs := make([]int64, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	rolesByUser := make(map[int64][]string)
	if len(userIDs) > 0 {
		roleRows, err := h.store.ListUserRolesByUserIDs(ctx, userIDs)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		for _, row := range roleRows {
			rolesByUser[row.UserID] = append(rolesByUser[row.UserID], row.RoleName)
		}
	}

	numUsers := len(users)
	items := make([]models.User, numUsers)
	for i, user := range users {
		items[i] = models.NewUser(user, rolesByUser[user.ID])
	}

	count, err := h.store.CountUsers(ctx, util.ToPgText(req.Status))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := util.NewPaginatedList(req.Page, req.PerPage, int32(count), items)

	ctx.JSON(http.StatusOK, res)
}

type getUserReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GetUser
//
//	@Summary	Get user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Router		/users/{id} [get]
func (h *DefaultHandler) GetUser(ctx *gin.Context) {
	var req getUserReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := h.store.GetUser(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	roleNames, _ := h.store.ListUserRoles(ctx, req.ID)

	ctx.JSON(http.StatusOK, models.NewUser(user, roleNames))
}

type updateUserUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateUserReq struct {
	Password string   `json:"password" binding:"omitempty,min=6"`
	FullName string   `json:"full_name" binding:"omitempty,fullname"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Roles    []string `json:"roles"`
} //@name UpdateUserParams

// UpdateUser
//
//	@Summary	Update user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int				true	"User ID"
//	@Param		req	body	updateUserReq	true	"Update user parameters"
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Router		/users/{id} [put]
func (h *DefaultHandler) UpdateUser(ctx *gin.Context) {
	var uri updateUserUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateUserReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if !isSuperAdminSession(ctx) && containsSuperAdminRole(req.Roles) {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("only a superadmin can grant the SUPERADMIN role")))
		return
	}

	var newPassword pgtype.Text
	var passwordChangedAt pgtype.Timestamptz
	if len(req.Password) > 0 {
		hashedPassword, err := util.HashPassword(req.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		newPassword.String = hashedPassword
		newPassword.Valid = true
		passwordChangedAt.Time = time.Now()
		passwordChangedAt.Valid = true
	}

	// Pre-flight the role change: authorize before mutating the user so a
	// rejected request never partially applies.
	var currentRoleNames []string
	if len(req.Roles) > 0 {
		roles, err := h.store.ListUserRoles(ctx, uri.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		currentRoleNames = roles

		// A non-superadmin may not touch a superadmin's roles at all (this also
		// stops them from stripping the SUPERADMIN role).
		if !isSuperAdminSession(ctx) && slices.Contains(currentRoleNames, string(models.SuperAdminRole)) {
			ctx.JSON(http.StatusForbidden, errorResponse(errors.New("only a superadmin can change a superadmin's roles")))
			return
		}
	}

	arg := db.UpdateUserParams{
		ID:                uri.ID,
		Password:          newPassword,
		PasswordChangedAt: passwordChangedAt,
		FullName: pgtype.Text{
			String: req.FullName,
			Valid:  len(req.FullName) > 0,
		},
		Email: pgtype.Text{
			String: req.Email,
			Valid:  len(req.Email) > 0,
		},
	}

	user, err := h.store.UpdateUser(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		} else if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(errors.New("email is already taken")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var updatedRoleNames []string
	if len(req.Roles) > 0 {
		reqRoleNames := make([]string, len(req.Roles))
		for r, roleName := range req.Roles {
			reqRoleNames[r] = strings.ToUpper(roleName)
		}

		updatedRoleNames = append(updatedRoleNames, util.SetIntersection(currentRoleNames, reqRoleNames)...)

		var createUserRolesArgs []db.UserRolesParams
		for _, roleName := range util.SetDifference(reqRoleNames, currentRoleNames) {
			createUserRolesArgs = append(createUserRolesArgs, db.UserRolesParams{
				RoleName: roleName,
				Username: user.Username,
			})
		}
		createdUserRoles, _ := h.store.BulkCreateUserRoles(ctx, createUserRolesArgs)
		for _, userRole := range createdUserRoles {
			updatedRoleNames = append(updatedRoleNames, userRole.RoleName)
		}

		var deleteUserRolesArgs []db.UserRolesParams
		for _, roleName := range util.SetDifference(currentRoleNames, reqRoleNames) {
			deleteUserRolesArgs = append(deleteUserRolesArgs, db.UserRolesParams{
				RoleName: roleName,
				Username: user.Username,
			})
		}
		h.store.BulkDeleteUserRoles(ctx, deleteUserRolesArgs)
	}

	ctx.JSON(http.StatusOK, models.NewUser(user, updatedRoleNames))
}

type deleteUserReq struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// DeleteUser
//
//	@Summary	Delete user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int	true	"User ID"
//	@Security	BearerAuth
//	@Success	204
//	@Router		/users/{id} [delete]
func (h *DefaultHandler) DeleteUser(ctx *gin.Context) {
	var req deleteUserReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := h.store.DeleteUser(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

type reviewUserUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type reviewUserReq struct {
	Note string `json:"note"`
} //@name ReviewUserParams

// ApproveUser
//
//	@Summary	Approve a pending registration
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int				true	"User ID"
//	@Param		req	body	reviewUserReq	false	"Review note"
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Router		/users/{id}/approve [post]
func (h *DefaultHandler) ApproveUser(ctx *gin.Context) {
	h.reviewUser(ctx, models.UserStatusActive)
}

// RejectUser
//
//	@Summary	Reject a pending registration
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id	path	int				true	"User ID"
//	@Param		req	body	reviewUserReq	false	"Review note"
//	@Security	BearerAuth
//	@Success	200	{object}	models.User
//	@Router		/users/{id}/reject [post]
func (h *DefaultHandler) RejectUser(ctx *gin.Context) {
	h.reviewUser(ctx, models.UserStatusRejected)
}

func (h *DefaultHandler) reviewUser(ctx *gin.Context, status string) {
	var uri reviewUserUri
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req reviewUserReq
	_ = ctx.ShouldBindJSON(&req)

	reviewer, ok := h.authUser(ctx)
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

	if user.Status != models.UserStatusPending {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("user is not pending approval")))
		return
	}

	row, err := h.store.ReviewUser(ctx, db.ReviewUserParams{
		ID:         uri.ID,
		Status:     status,
		ReviewedBy: pgtype.Int8{Int64: reviewer.ID, Valid: true},
		ReviewNote: util.ToPgText(req.Note),
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	roleNames, _ := h.store.ListUserRoles(ctx, row.ID)
	ctx.JSON(http.StatusOK, models.NewUser(row, roleNames))
}

type registerUserReq struct {
	Username        string `json:"username" binding:"required,alphanum"`
	Password        string `json:"password" binding:"required,min=6,eqfield=ConfirmPassword"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=6"`
	FullName        string `json:"full_name" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
} //@name RegisterUserParams

// RegisterUser
//
//	@Summary	Register user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	registerUserReq	true	"Register user parameters"
//	@Success	204
//	@Router		/users/register [post]
func (h *DefaultHandler) RegisterUser(ctx *gin.Context) {
	var req registerUserReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username: req.Username,
		Password: hashedPassword,
		FullName: req.FullName,
		Email:    req.Email,
		Status:   models.UserStatusPending,
	}

	user, err := h.store.CreateUser(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(errors.New("username or email is already taken")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	roleNames := []string{"USER"}
	createUserRolesArgs := []db.UserRolesParams{
		{
			Username: user.Username,
			RoleName: roleNames[0],
		},
	}

	h.store.BulkCreateUserRoles(ctx, createUserRolesArgs)

	ctx.JSON(http.StatusNoContent, nil)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
} //@name LoginUserParams

// LoginUser
//
//	@Summary	User login
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		req	body	loginUserRequest	true	"Login user parameters"
//	@Success	204
//	@Router		/users/login [post]
func (h *DefaultHandler) LoginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := h.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Self-registered users must be approved before they can log in.
	if user.Status != models.UserStatusActive {
		msg := "account is not active"
		switch user.Status {
		case models.UserStatusPending:
			msg = "account is pending approval"
		case models.UserStatusRejected:
			msg = "account registration was rejected"
		}
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New(msg)))
		return
	}

	roleNames, _ := h.store.ListUserRoles(ctx, user.ID)

	payloadUser := token.User{
		Username: user.Username,
		Roles:    roleNames,
	}

	// The refresh token defines the session; the access token is bound to it
	// so that deleting the session (logout/block) revokes the access token too.
	refreshToken, refreshPayload, err := h.tokenMaker.CreateToken(payloadUser, uuid.Nil, h.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_, err = h.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: util.HashToken(refreshToken),
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    pgtype.Timestamptz{Time: refreshPayload.ExpiresAt, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := h.tokenMaker.CreateToken(payloadUser, refreshPayload.ID, h.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	h.setAuthCookie(ctx, models.AccessTokenCookieName, accessToken, accessPayload.ExpiresAt)
	h.setAuthCookie(ctx, models.RefreshTokenCookieName, refreshToken, refreshPayload.ExpiresAt)

	ctx.JSON(http.StatusNoContent, nil)
}

// LogoutUser
//
//	@Summary	User logout
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Success	204
//	@Router		/users/logout [post]
func (h *DefaultHandler) LogoutUser(ctx *gin.Context) {
	refreshToken, _ := ctx.Cookie(models.RefreshTokenCookieName)

	if refreshPayload, err := h.tokenMaker.VerifyToken(refreshToken); err == nil && refreshPayload != nil {
		h.store.DeleteSession(ctx, refreshPayload.ID)
		// Drop the cached liveness so the revocation is immediate.
		session.Invalidate(refreshPayload.ID)
	}

	h.clearAuthCookie(ctx, models.AccessTokenCookieName)
	h.clearAuthCookie(ctx, models.RefreshTokenCookieName)

	ctx.JSON(http.StatusNoContent, nil)
}

// GetAuthUser
//
//	@Summary	Get Auth User
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	User
//	@Router		/users/auth [get]
func (h *DefaultHandler) GetAuthUser(ctx *gin.Context) {
	payload, _ := ctx.Get(models.AuthPayloadKey)
	authPayload, _ := payload.(*token.Payload)

	user, err := h.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, models.NewUser(user, authPayload.User.Roles))
}
