package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserReq struct {
	Username string   `json:"username" binding:"required,alphanum"`
	Password string   `json:"password" binding:"required,min=6"`
	FullName string   `json:"full_name" binding:"required,alphanumspace"`
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
func (s *Server) CreateUser(ctx *gin.Context) {
	var req createUserReq
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
	}

	user, err := s.store.CreateUser(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
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
		userRoles, _ := s.store.BulkCreateUserRoles(ctx, createUserRolesArgs)

		for _, userRole := range userRoles {
			roleNames = append(roleNames, userRole.RoleName)
		}
	}

	ctx.JSON(http.StatusOK, models.NewUser(user, roleNames))
}

type listUsersReq struct {
	Page    int32 `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int32 `form:"per_page,default=5" binding:"omitempty,min=1,max=30"`
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
func (s *Server) ListUsers(ctx *gin.Context) {
	var req listUsersReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.Page - 1) * req.PerPage
	arg := db.ListUsersParams{
		Limit:  req.PerPage,
		Offset: offset,
	}
	users, err := s.store.ListUsers(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	numUsers := len(users)
	items := make([]models.User, numUsers)
	for i, user := range users {
		items[i] = models.NewUser(user, nil)
	}

	count, err := s.store.CountUsers(ctx)
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
func (s *Server) GetUser(ctx *gin.Context) {
	var req getUserReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := s.store.GetUser(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	roleNames, _ := s.store.ListUserRoles(ctx, req.ID)

	ctx.JSON(http.StatusOK, models.NewUser(user, roleNames))
}

type updateUserUri struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateUserReq struct {
	Password string   `json:"password" binding:"omitempty,min=6"`
	FullName string   `json:"full_name" binding:"omitempty,alphanumspace"`
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
func (s *Server) UpdateUser(ctx *gin.Context) {
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

	user, err := s.store.UpdateUser(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		} else if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var updatedRoleNames []string
	if len(req.Roles) > 0 {
		roleNames, err := s.store.ListUserRoles(ctx, uri.ID)
		if err == nil {
			reqRoleNames := make([]string, len(req.Roles))
			for r, roleName := range req.Roles {
				reqRoleNames[r] = strings.ToUpper(roleName)
			}

			updatedRoleNames = append(updatedRoleNames, util.SetIntersection(roleNames, reqRoleNames)...)

			var createUserRolesArgs []db.UserRolesParams
			for _, roleName := range util.SetDifference(reqRoleNames, roleNames) {
				createUserRolesArgs = append(createUserRolesArgs, db.UserRolesParams{
					RoleName: roleName,
					Username: user.Username,
				})
			}
			createdUserRoles, _ := s.store.BulkCreateUserRoles(ctx, createUserRolesArgs)
			for _, userRole := range createdUserRoles {
				updatedRoleNames = append(updatedRoleNames, userRole.RoleName)
			}

			var deleteUserRolesArgs []db.UserRolesParams
			for _, roleName := range util.SetDifference(roleNames, reqRoleNames) {
				deleteUserRolesArgs = append(deleteUserRolesArgs, db.UserRolesParams{
					RoleName: roleName,
					Username: user.Username,
				})
			}
			s.store.BulkDeleteUserRoles(ctx, deleteUserRolesArgs)
		}
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
func (s *Server) DeleteUser(ctx *gin.Context) {
	var req deleteUserReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := s.store.DeleteUser(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
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
func (s *Server) RegisterUser(ctx *gin.Context) {
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
	}

	user, err := s.store.CreateUser(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
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

	s.store.BulkCreateUserRoles(ctx, createUserRolesArgs)

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
func (s *Server) LoginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := s.store.GetUserByUsername(ctx, req.Username)
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

	roleNames, _ := s.store.ListUserRoles(ctx, user.ID)

	payloadUser := token.User{
		Username: user.Username,
		Roles:    roleNames,
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(payloadUser, s.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(payloadUser, s.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	_, err = s.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    pgtype.Timestamptz{Time: refreshPayload.ExpiresAt, Valid: true},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	cookieIsSecure := s.config.Environment == "production"

	ctx.SetCookie(accessTokenCookieName, accessToken, int(time.Until(accessPayload.ExpiresAt).Seconds()), s.config.CookiePath, s.config.CookieDomain, cookieIsSecure, true)
	ctx.SetCookie(refreshTokenCookieName, refreshToken, int(time.Until(refreshPayload.ExpiresAt).Seconds()), s.config.CookiePath, s.config.CookieDomain, cookieIsSecure, true)

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
func (s *Server) LogoutUser(ctx *gin.Context) {
	refreshToken, _ := ctx.Cookie(refreshTokenCookieName)

	refreshPayload, _ := s.tokenMaker.VerifyToken(refreshToken)

	s.store.DeleteSession(ctx, refreshPayload.ID)
	ctx.SetCookie(accessTokenCookieName, "", -1, s.config.CookiePath, s.config.CookieDomain, true, true)
	ctx.SetCookie(refreshTokenCookieName, "", -1, s.config.CookiePath, s.config.CookieDomain, true, true)

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
func (s *Server) GetAuthUser(ctx *gin.Context) {
	payload, _ := ctx.Get(authPayloadKey)
	authPayload, _ := payload.(*token.Payload)

	user, err := s.store.GetUserByUsername(ctx, authPayload.User.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, models.NewUser(user, authPayload.User.Roles))
}
