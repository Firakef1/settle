package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/oauth"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

var (
	ErrOAuthProviderUnavailable = errors.New("sign-in provider is not configured")
	ErrOAuthAccountExists       = errors.New("an account with this email already exists; sign in with your password")
)

// OAuthService signs people in with an external provider.
type OAuthService struct {
	providers    map[string]*oauth.Provider
	userRepo     repository.UserRepository
	identityRepo repository.IdentityRepository
	authService  *AuthService
	hashService  *sharedService.HashService
}

// NewOAuthService creates an OAuthService for the configured providers.
func NewOAuthService(
	providers map[string]*oauth.Provider,
	userRepo repository.UserRepository,
	identityRepo repository.IdentityRepository,
	authService *AuthService,
	hashService *sharedService.HashService,
) *OAuthService {
	return &OAuthService{providers: providers, userRepo: userRepo, identityRepo: identityRepo, authService: authService, hashService: hashService}
}

// Enabled lists the providers that have credentials, sorted.
func (s *OAuthService) Enabled() []string {
	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Provider returns a configured provider by name.
func (s *OAuthService) Provider(name string) (*oauth.Provider, error) {
	p, ok := s.providers[name]
	if !ok {
		return nil, ErrOAuthProviderUnavailable
	}
	return p, nil
}

// SignIn finds or creates the user for a provider profile and issues a session.
//
//  1. A provider account already linked to a user signs in as that user.
//  2. Otherwise an existing account with the same email is linked, but only
//     when the provider has verified the email (Google does; Microsoft
//     doesn't say), so nobody can take over an account by claiming its email.
//     An unverified local account is verified and its password replaced, so
//     whoever registered the address without proving it can't keep access.
//  3. Otherwise a new, verified account is created. It has no usable password
//     until the user sets one with "Forgot password".
func (s *OAuthService) SignIn(ctx context.Context, provider string, profile oauth.Profile) (*dto.LoginResponse, error) {
	identity, err := s.identityRepo.FindByProviderSubject(ctx, provider, profile.Subject)
	if err == nil {
		user, err := s.userRepo.FindByID(ctx, identity.UserID)
		if err != nil {
			return nil, err
		}
		return s.authService.IssueSession(ctx, user)
	}
	if !errors.Is(err, repository.ErrIdentityNotFound) {
		return nil, err
	}

	email := normalizeEmail(profile.Email)
	user, err := s.userRepo.FindByEmail(ctx, email)
	switch {
	case err == nil:
		if !profile.EmailVerified {
			return nil, ErrOAuthAccountExists
		}
		if !user.EmailVerified {
			if err := s.replacePassword(ctx, user.ID); err != nil {
				return nil, err
			}
			if err := s.userRepo.MarkEmailAsVerified(ctx, user.Email); err != nil && !errors.Is(err, repository.ErrUserNotFound) {
				return nil, err
			}
			user.EmailVerified = true
		}
	case errors.Is(err, repository.ErrUserNotFound):
		user, err = s.createUser(ctx, email, profile.Name)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	if err := s.identityRepo.Create(ctx, &model.UserIdentity{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Provider:  provider,
		Subject:   profile.Subject,
		Email:     email,
		CreatedAt: time.Now(),
	}); err != nil && !errors.Is(err, repository.ErrIdentityAlreadyExists) {
		return nil, err
	}
	return s.authService.IssueSession(ctx, user)
}

func (s *OAuthService) unusablePasswordHash() (string, error) {
	random, err := generateRefreshToken()
	if err != nil {
		return "", err
	}
	return s.hashService.Hash(random)
}

func (s *OAuthService) replacePassword(ctx context.Context, userID string) error {
	hash, err := s.unusablePasswordHash()
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, hash)
}

func (s *OAuthService) createUser(ctx context.Context, email, name string) (*model.User, error) {
	hash, err := s.unusablePasswordHash()
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	now := time.Now()
	user := &model.User{
		ID:            uuid.New().String(),
		Email:         email,
		Name:          name,
		PasswordHash:  hash,
		Status:        "active",
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			// e.g. a deleted account still holds the address
			return nil, ErrOAuthAccountExists
		}
		return nil, err
	}
	return user, nil
}
