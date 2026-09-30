package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

func TestPasswordResetOTPRepo_Memory(t *testing.T) {
	repo := NewPasswordResetOTPRepo(nil)
	ctx := context.Background()

	userID := "user-123"
	otp := &model.PasswordResetOTP{
		ID:        "otp-1",
		UserID:    userID,
		OTPHash:   "hashed-otp",
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 1. Upsert
	err := repo.UpsertOTP(ctx, otp)
	assert.NoError(t, err)

	// 2. Find
	found, err := repo.FindByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, "hashed-otp", found.OTPHash)
	assert.Equal(t, 0, found.AttemptCount)

	// 3. Increment Attempt
	count, err := repo.IncrementAttemptCount(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	found2, err := repo.FindByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 1, found2.AttemptCount)

	// 4. Delete
	err = repo.DeleteByUserID(ctx, userID)
	assert.NoError(t, err)

	_, err = repo.FindByUserID(ctx, userID)
	assert.Equal(t, ErrOTPNotFound, err)
}
