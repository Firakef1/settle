package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

func TestVerificationCodeRepo_SaveAndFind(t *testing.T) {
	ctx := context.Background()
	repo := NewVerificationCodeRepo(nil)

	now := time.Now()
	code := &model.VerificationCode{
		ID:        "vc-1",
		Email:     "user@example.com",
		Code:      "123456",
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
	}

	err := repo.Save(ctx, code)
	require.NoError(t, err)

	// Find by email and code
	found, err := repo.FindByEmailAndCode(ctx, "USER@EXAMPLE.COM", "123456")
	require.NoError(t, err)
	assert.Equal(t, "vc-1", found.ID)
	assert.Equal(t, "123456", found.Code)

	// Wrong code -> ErrVerificationCodeNotFound
	_, err = repo.FindByEmailAndCode(ctx, "user@example.com", "000000")
	assert.ErrorIs(t, err, ErrVerificationCodeNotFound)

	// Delete by email
	err = repo.DeleteByEmail(ctx, "user@example.com")
	require.NoError(t, err)

	_, err = repo.FindByEmailAndCode(ctx, "user@example.com", "123456")
	assert.ErrorIs(t, err, ErrVerificationCodeNotFound)
}

func TestVerificationCodeRepo_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	repo := NewVerificationCodeRepo(nil)

	now := time.Now()
	expired := &model.VerificationCode{
		ID:        "vc-expired",
		Email:     "expired@example.com",
		Code:      "654321",
		ExpiresAt: now.Add(-5 * time.Minute),
		CreatedAt: now.Add(-20 * time.Minute),
	}
	err := repo.Save(ctx, expired)
	require.NoError(t, err)

	count, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
