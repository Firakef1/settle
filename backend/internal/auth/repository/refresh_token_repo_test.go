package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

func TestRefreshTokenRepo_Store_Find(t *testing.T) {
	repo := NewRefreshTokenRepo(nil)

	token := &model.RefreshToken{
		ID:        "token1",
		UserID:    "user1",
		TokenHash: "hash123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.StoreRefreshToken(context.Background(), token)
	require.NoError(t, err)

	found, err := repo.FindByTokenHash(context.Background(), "hash123")
	require.NoError(t, err)
	assert.Equal(t, "token1", found.ID)
	assert.Equal(t, "user1", found.UserID)
	assert.False(t, found.Revoked)
}

func TestRefreshTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	repo := NewRefreshTokenRepo(nil)

	_, err := repo.FindByTokenHash(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestRefreshTokenRepo_RevokeByTokenHash(t *testing.T) {
	repo := NewRefreshTokenRepo(nil)

	token := &model.RefreshToken{
		ID:        "token1",
		UserID:    "user1",
		TokenHash: "hash123",
	}

	err := repo.StoreRefreshToken(context.Background(), token)
	require.NoError(t, err)

	err = repo.RevokeByTokenHash(context.Background(), "hash123")
	require.NoError(t, err)

	found, err := repo.FindByTokenHash(context.Background(), "hash123")
	require.NoError(t, err)
	assert.True(t, found.Revoked)
}

func TestRefreshTokenRepo_RevokeAllForUser(t *testing.T) {
	repo := NewRefreshTokenRepo(nil)

	token1 := &model.RefreshToken{
		ID:        "token1",
		UserID:    "user1",
		TokenHash: "hash1",
	}
	repo.StoreRefreshToken(context.Background(), token1)

	token2 := &model.RefreshToken{
		ID:        "token2",
		UserID:    "user1",
		TokenHash: "hash2",
	}
	repo.StoreRefreshToken(context.Background(), token2)

	token3 := &model.RefreshToken{
		ID:        "token3",
		UserID:    "user2",
		TokenHash: "hash3",
	}
	repo.StoreRefreshToken(context.Background(), token3)

	err := repo.RevokeAllForUser(context.Background(), "user1")
	require.NoError(t, err)

	found1, _ := repo.FindByTokenHash(context.Background(), "hash1")
	assert.True(t, found1.Revoked)

	found2, _ := repo.FindByTokenHash(context.Background(), "hash2")
	assert.True(t, found2.Revoked)

	found3, _ := repo.FindByTokenHash(context.Background(), "hash3")
	assert.False(t, found3.Revoked)
}

func TestRefreshTokenRepo_RevokeByTokenHash_NotFound(t *testing.T) {
	repo := NewRefreshTokenRepo(nil)

	err := repo.RevokeByTokenHash(context.Background(), "nonexistent")
	assert.Error(t, err)
}
