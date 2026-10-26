package db

import (
	"context"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestUserEmailIsUniqueCaseInsensitively(t *testing.T) {
	require.NoError(t, testMigration.Up())
	defer func() {
		require.NoError(t, testMigration.Down())
	}()

	ctx := context.Background()
	hashedPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	create := func(email string) error {
		_, err := testStore.CreateUser(ctx, CreateUserParams{
			Username: util.RandomString(12),
			Password: hashedPassword,
			FullName: "Email Test",
			Email:    pgtype.Text{String: email, Valid: true},
			Status:   "active",
		})
		return err
	}

	require.NoError(t, create("Case@Test.Example"))
	require.Equal(t, UniqueViolation, ErrorCode(create("case@test.example")))
}

func TestRegistrationTokenLifecycle(t *testing.T) {
	require.NoError(t, testMigration.Up())
	defer func() {
		require.NoError(t, testMigration.Down())
	}()

	ctx := context.Background()

	hashedPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	user, err := testStore.CreateUser(ctx, CreateUserParams{
		Username: util.RandomString(12),
		Password: hashedPassword,
		FullName: "Invited User",
		Email:    pgtype.Text{},
		Status:   "invited",
	})
	require.NoError(t, err)

	firstRaw := util.RandomString(32)
	first, err := testStore.CreateRegistrationTokenTx(ctx, CreateRegistrationTokenTxParams{
		UserID:    user.ID,
		TokenHash: util.HashToken(firstRaw),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.NotZero(t, first.ID)

	// Regenerating invalidates the previous unused link.
	secondRaw := util.RandomString(32)
	_, err = testStore.CreateRegistrationTokenTx(ctx, CreateRegistrationTokenTxParams{
		UserID:    user.ID,
		TokenHash: util.HashToken(secondRaw),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	stale, err := testStore.GetUserTokenByHash(ctx, GetUserTokenByHashParams{
		TokenHash: util.HashToken(firstRaw),
		Purpose:   UserTokenPurposeRegistration,
	})
	require.NoError(t, err)
	require.True(t, stale.UsedAt.Valid)

	_, err = testStore.CompleteRegistrationTx(ctx, CompleteRegistrationTxParams{
		TokenHash: util.HashToken(firstRaw),
		Password:  hashedPassword,
	})
	require.ErrorIs(t, err, ErrRecordNotFound)

	newPassword, err := util.HashPassword(util.RandomString(12))
	require.NoError(t, err)

	activated, err := testStore.CompleteRegistrationTx(ctx, CompleteRegistrationTxParams{
		TokenHash: util.HashToken(secondRaw),
		Password:  newPassword,
		FullName:  pgtype.Text{String: "Activated User", Valid: true},
		Email:     pgtype.Text{String: "activated@example.com", Valid: true},
		SetEmail:  true,
	})
	require.NoError(t, err)
	require.Equal(t, "active", activated.Status)
	require.Equal(t, "Activated User", activated.FullName)
	require.Equal(t, "activated@example.com", activated.Email.String)
	require.True(t, activated.PasswordChangedAt.Valid)
	require.Equal(t, newPassword, activated.Password)

	// A consumed token cannot be reused.
	_, err = testStore.CompleteRegistrationTx(ctx, CompleteRegistrationTxParams{
		TokenHash: util.HashToken(secondRaw),
		Password:  newPassword,
	})
	require.ErrorIs(t, err, ErrRecordNotFound)
}
