package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

func TestUserRepo_CreateUser_Success(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:           "user1",
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "hash",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err)

	found, err := repo.FindByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user1", found.ID)
}

func TestUserRepo_CreateUser_Duplicate(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:    "user1",
		Email: "test@example.com",
	}

	err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err)

	err = repo.CreateUser(context.Background(), user)
	assert.Error(t, err)
}

func TestUserRepo_FindByEmail_Found(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:     "user1",
		Email:  "test@example.com",
		Status: "active",
	}
	_ = repo.CreateUser(context.Background(), user)

	found, err := repo.FindByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user1", found.ID)
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	repo := NewUserRepo(nil)
	_, err := repo.FindByEmail(context.Background(), "nonexistent@example.com")
	assert.Error(t, err)
}

func TestUserRepo_FindByEmail_CaseInsensitive(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:     "user1",
		Email:  "Test@Example.COM",
		Status: "active",
	}
	_ = repo.CreateUser(context.Background(), user)

	found, err := repo.FindByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user1", found.ID)
}

func TestUserRepo_FindByID_Found(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:     "user1",
		Email:  "test@example.com",
		Status: "active",
	}
	_ = repo.CreateUser(context.Background(), user)

	found, err := repo.FindByID(context.Background(), "user1")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", found.Email)
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	repo := NewUserRepo(nil)
	_, err := repo.FindByID(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestUserRepo_GetOrgMemberships_Empty(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:    "user1",
		Email: "test@example.com",
	}
	_ = repo.CreateUser(context.Background(), user)

	memberships, err := repo.GetOrgMemberships(context.Background(), "user1")
	require.NoError(t, err)
	assert.Empty(t, memberships)
}

func TestUserRepo_GetOrgMemberships_WithData(t *testing.T) {
	repo := NewUserRepo(nil)
	user := &model.User{
		ID:    "user1",
		Email: "test@example.com",
	}
	_ = repo.CreateUser(context.Background(), user)

	membership := model.OrgMembership{
		ID:      "mem1",
		OrgID:   "org1",
		OrgName: "Test Org",
		OrgSlug: "test-org",
		UserID:  "user1",
		Role:    "admin",
		Status:  "active",
	}
	repo.AddMemoryOrgMembership(membership)

	memberships, err := repo.GetOrgMemberships(context.Background(), "user1")
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	assert.Equal(t, "org1", memberships[0].OrgID)
}
