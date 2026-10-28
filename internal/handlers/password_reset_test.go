package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	mockdb "github.com/emiliogozo/panahon-api-go/internal/mocks/db"
	mocktoken "github.com/emiliogozo/panahon-api-go/internal/mocks/token"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newPasswordResetContext(t *testing.T, method, path, body string, userID int64) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	if userID > 0 {
		ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", userID)}}
	}
	return ctx, recorder
}

func TestCreatePasswordResetLinkAPI(t *testing.T) {
	admin, _, _ := randomUser(t)
	user, _, _ := randomUser(t)

	t.Run("ActiveUser", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/1/password-reset-link", "", user.ID)
		ctx.Set(models.AuthPayloadKey, &token.Payload{User: token.User{Username: admin.Username}})

		store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), admin.Username).
			Return(admin, nil)
		store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
			Return(user, nil)
		store.EXPECT().CreatePasswordResetTokenTx(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.CreatePasswordResetTokenTxParams) bool {
			return arg.UserID == user.ID &&
				arg.TokenHash != "" &&
				arg.CreatedBy.Valid && arg.CreatedBy.Int64 == admin.ID &&
				arg.ExpiresAt.After(time.Now())
		})).Return(db.UserToken{}, nil)

		handler.CreatePasswordResetLink(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

		var res passwordResetLinkRes
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
		require.True(t, strings.HasPrefix(res.ResetURL, "http://localhost:8000/reset-password?token="))
	})

	t.Run("InactiveUser", func(t *testing.T) {
		invited := user
		invited.Status = models.UserStatusInvited

		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/1/password-reset-link", "", invited.ID)
		ctx.Set(models.AuthPayloadKey, &token.Payload{User: token.User{Username: admin.Username}})

		store.EXPECT().GetUserByUsername(mock.AnythingOfType("*gin.Context"), admin.Username).
			Return(admin, nil)
		store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), invited.ID).
			Return(invited, nil)

		handler.CreatePasswordResetLink(ctx)

		require.Equal(t, http.StatusConflict, recorder.Code)
	})
}

func TestValidatePasswordResetAPI(t *testing.T) {
	user, _, _ := randomUser(t)
	rawToken := util.RandomString(32)

	validToken := db.UserToken{
		ID:        1,
		UserID:    user.ID,
		Purpose:   db.UserTokenPurposePasswordReset,
		TokenHash: util.HashToken(rawToken),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}

	t.Run("Valid", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/validate",
			fmt.Sprintf(`{"token":%q}`, rawToken), 0)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), db.GetUserTokenByHashParams{
			TokenHash: util.HashToken(rawToken),
			Purpose:   db.UserTokenPurposePasswordReset,
		}).Return(validToken, nil)
		store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
			Return(user, nil)

		handler.ValidatePasswordReset(ctx)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

		var res passwordResetInfoRes
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
		require.Equal(t, user.Username, res.Username)
		require.Equal(t, user.FullName, res.FullName)
		require.Equal(t, user.Email.String, res.Email)
	})

	t.Run("Expired", func(t *testing.T) {
		expired := validToken
		expired.ExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}

		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/validate",
			fmt.Sprintf(`{"token":%q}`, rawToken), 0)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(expired, nil)

		handler.ValidatePasswordReset(ctx)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("UnknownToken", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/validate",
			fmt.Sprintf(`{"token":%q}`, rawToken), 0)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(db.UserToken{}, db.ErrRecordNotFound)

		handler.ValidatePasswordReset(ctx)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("InactiveUser", func(t *testing.T) {
		invited := user
		invited.Status = models.UserStatusInvited

		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)
		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/validate",
			fmt.Sprintf(`{"token":%q}`, rawToken), 0)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(validToken, nil)
		store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
			Return(invited, nil)

		handler.ValidatePasswordReset(ctx)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestAcceptPasswordResetAPI(t *testing.T) {
	user, _, _ := randomUser(t)
	rawToken := util.RandomString(32)

	validToken := db.UserToken{
		ID:        1,
		UserID:    user.ID,
		Purpose:   db.UserTokenPurposePasswordReset,
		TokenHash: util.HashToken(rawToken),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}

	t.Run("OK", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		tokenMaker := mocktoken.NewMockMaker(t)
		handler := newTestHandler(store, tokenMaker)

		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/accept",
			fmt.Sprintf(`{"token":%q,"password":"newpassword","confirm_password":"newpassword"}`, rawToken), 0)

		tokenStr := util.RandomString(32)
		tokenExpiresAt := time.Now().Add(time.Hour)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(validToken, nil)
		store.EXPECT().GetUser(mock.AnythingOfType("*gin.Context"), user.ID).
			Return(user, nil)
		store.EXPECT().ResetPasswordTx(mock.AnythingOfType("*gin.Context"), mock.MatchedBy(func(arg db.ResetPasswordTxParams) bool {
			return arg.TokenHash == util.HashToken(rawToken) && arg.Password != ""
		})).Return(user, nil)
		store.EXPECT().ListUserRoles(mock.AnythingOfType("*gin.Context"), user.ID).
			Return([]string{}, nil)
		tokenMaker.EXPECT().CreateToken(mock.Anything, mock.Anything, mock.Anything).
			Return(tokenStr, &token.Payload{ExpiresAt: tokenExpiresAt}, nil).Twice()
		store.EXPECT().CreateSession(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(db.Session{}, nil)

		handler.AcceptPasswordReset(ctx)

		require.Equal(t, http.StatusNoContent, recorder.Code)
		require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

		cookies := recorder.Result().Cookies()
		require.NotEmpty(t, cookies)
	})

	t.Run("ShortPassword", func(t *testing.T) {
		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)

		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/accept",
			fmt.Sprintf(`{"token":%q,"password":"short","confirm_password":"short"}`, rawToken), 0)

		handler.AcceptPasswordReset(ctx)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		expired := validToken
		expired.ExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}

		store := mockdb.NewMockStore(t)
		handler := newTestHandler(store, nil)

		ctx, recorder := newPasswordResetContext(t, http.MethodPost, "/users/password-reset/accept",
			fmt.Sprintf(`{"token":%q,"password":"newpassword","confirm_password":"newpassword"}`, rawToken), 0)

		store.EXPECT().GetUserTokenByHash(mock.AnythingOfType("*gin.Context"), mock.Anything).
			Return(expired, nil)

		handler.AcceptPasswordReset(ctx)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}
