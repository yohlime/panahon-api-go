package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	UserTokenPurposeRegistration  = "registration"
	UserTokenPurposePasswordReset = "password_reset"
)

type CreateRegistrationTokenTxParams struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedBy pgtype.Int8
}

// CreateRegistrationTokenTx invalidates any unused registration tokens for the
// user and creates a new one atomically.
func (store *SQLStore) CreateRegistrationTokenTx(ctx context.Context, arg CreateRegistrationTokenTxParams) (UserToken, error) {
	var token UserToken

	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.InvalidateUserTokens(ctx, InvalidateUserTokensParams{
			UserID:  arg.UserID,
			Purpose: UserTokenPurposeRegistration,
		}); err != nil {
			return err
		}

		var err error
		token, err = q.CreateUserToken(ctx, CreateUserTokenParams{
			UserID:    arg.UserID,
			Purpose:   UserTokenPurposeRegistration,
			TokenHash: arg.TokenHash,
			ExpiresAt: pgtype.Timestamptz{Time: arg.ExpiresAt, Valid: true},
			CreatedBy: arg.CreatedBy,
		})
		return err
	})

	return token, err
}

type CompleteRegistrationTxParams struct {
	TokenHash string
	Password  string
	FullName  pgtype.Text
	Email     pgtype.Text
	SetEmail  bool
}

// CompleteRegistrationTx consumes a registration token and activates the
// invited user in one transaction. If the user was already activated (or the
// email conflicts), the whole transaction rolls back and the token stays usable.
func (store *SQLStore) CompleteRegistrationTx(ctx context.Context, arg CompleteRegistrationTxParams) (User, error) {
	var user User

	err := store.execTx(ctx, func(q *Queries) error {
		token, err := q.ConsumeUserToken(ctx, ConsumeUserTokenParams{
			TokenHash: arg.TokenHash,
			Purpose:   UserTokenPurposeRegistration,
		})
		if err != nil {
			return err
		}

		user, err = q.CompleteUserRegistration(ctx, CompleteUserRegistrationParams{
			ID:       token.UserID,
			Password: arg.Password,
			FullName: arg.FullName,
			Email:    arg.Email,
			SetEmail: arg.SetEmail,
		})
		return err
	})

	return user, err
}

type CreatePasswordResetTokenTxParams struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedBy pgtype.Int8
}

// CreatePasswordResetTokenTx invalidates any unused reset tokens for the user
// and creates a new one atomically.
func (store *SQLStore) CreatePasswordResetTokenTx(ctx context.Context, arg CreatePasswordResetTokenTxParams) (UserToken, error) {
	var token UserToken

	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.InvalidateUserTokens(ctx, InvalidateUserTokensParams{
			UserID:  arg.UserID,
			Purpose: UserTokenPurposePasswordReset,
		}); err != nil {
			return err
		}

		var err error
		token, err = q.CreateUserToken(ctx, CreateUserTokenParams{
			UserID:    arg.UserID,
			Purpose:   UserTokenPurposePasswordReset,
			TokenHash: arg.TokenHash,
			ExpiresAt: pgtype.Timestamptz{Time: arg.ExpiresAt, Valid: true},
			CreatedBy: arg.CreatedBy,
		})
		return err
	})

	return token, err
}

type ResetPasswordTxParams struct {
	TokenHash string
	Password  string
}

// ResetPasswordTx consumes a reset token, changes the password and revokes all
// existing sessions atomically.
func (store *SQLStore) ResetPasswordTx(ctx context.Context, arg ResetPasswordTxParams) (User, error) {
	var user User

	err := store.execTx(ctx, func(q *Queries) error {
		token, err := q.ConsumeUserToken(ctx, ConsumeUserTokenParams{
			TokenHash: arg.TokenHash,
			Purpose:   UserTokenPurposePasswordReset,
		})
		if err != nil {
			return err
		}

		user, err = q.UpdateUser(ctx, UpdateUserParams{
			ID:                token.UserID,
			Password:          pgtype.Text{String: arg.Password, Valid: true},
			PasswordChangedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}

		return q.DeleteSessionsByUserID(ctx, token.UserID)
	})

	return user, err
}
