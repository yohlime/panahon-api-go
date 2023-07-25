package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessTokenExpiresAt pgtype.Timestamptz `json:"access_token_expires_at"`
	AccessToken          string             `json:"access_token"`
}

func (s *Server) RenewAccessToken(c *gin.Context) {
	var req renewAccessTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	refreshPayload, err := s.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	session, err := s.store.GetSession(c, refreshPayload.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if session.IsBlocked {
		err := fmt.Errorf("blocked session")
		c.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	user, err := s.store.GetUser(c, session.UserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if user.Username != refreshPayload.User.Username {
		err := fmt.Errorf("incorrect session user")
		c.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.RefreshToken != req.RefreshToken {
		err := fmt.Errorf("mismatched session token")
		c.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if time.Now().After(session.ExpiresAt.Time) {
		err := fmt.Errorf("expired session")
		c.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(refreshPayload.User, s.config.AccessTokenDuration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: pgtype.Timestamptz{Time: accessPayload.ExpiredAt, Valid: true},
	}
	c.JSON(http.StatusOK, rsp)
}
