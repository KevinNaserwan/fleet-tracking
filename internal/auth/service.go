package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Result struct {
	Tokens TokenPair
	User   CurrentUser
}

type Service struct {
	repository Repository
	tokens     *TokenManager
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(repository Repository, tokens *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{repository: repository, tokens: tokens, refreshTTL: refreshTTL, now: time.Now}
}

func (s *Service) Login(ctx context.Context, input LoginInput, client ClientMetadata) (Result, error) {
	user, err := s.repository.FindForLogin(ctx, input.TenantSlug, input.Email)
	if errors.Is(err, ErrNotFound) {
		BurnPassword(input.Password)
		return Result{}, ErrUnauthorized
	}
	if err != nil {
		return Result{}, fmt.Errorf("find login identity: %w", err)
	}
	valid, err := VerifyPassword(input.Password, user.PasswordHash)
	if err != nil {
		return Result{}, fmt.Errorf("verify stored password: %w", err)
	}
	if !valid {
		return Result{}, ErrUnauthorized
	}

	rawRefresh, refreshHash, err := NewRefreshToken(user.Tenant.ID)
	if err != nil {
		return Result{}, err
	}
	session := NewSession{
		ID:        uuid.NewString(),
		TenantID:  user.Tenant.ID,
		UserID:    user.ID,
		FamilyID:  uuid.NewString(),
		TokenHash: refreshHash,
		ExpiresAt: s.now().UTC().Add(s.refreshTTL),
		Client:    client,
	}
	if err := s.repository.CreateSession(ctx, session); err != nil {
		return Result{}, fmt.Errorf("create refresh session: %w", err)
	}
	access, err := s.tokens.Sign(user)
	if err != nil {
		return Result{}, fmt.Errorf("sign access token: %w", err)
	}
	return Result{
		Tokens: TokenPair{
			AccessToken: access, RefreshToken: rawRefresh, TokenType: "Bearer",
			ExpiresIn: int64(s.tokens.AccessTTL() / time.Second),
		},
		User: user.Current(),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefresh string, client ClientMetadata) (TokenPair, error) {
	tenantID, currentHash, err := ParseRefreshToken(rawRefresh)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}
	newRaw, newHash, err := NewRefreshToken(tenantID)
	if err != nil {
		return TokenPair{}, err
	}
	user, err := s.repository.RotateSession(ctx, tenantID, currentHash, RotationInput{
		ID:        uuid.NewString(),
		TokenHash: newHash,
		ExpiresAt: s.now().UTC().Add(s.refreshTTL),
		Client:    client,
	})
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrRefreshReused) {
		return TokenPair{}, ErrUnauthorized
	}
	if err != nil {
		return TokenPair{}, fmt.Errorf("rotate refresh session: %w", err)
	}
	access, err := s.tokens.Sign(user)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	return TokenPair{
		AccessToken: access, RefreshToken: newRaw, TokenType: "Bearer",
		ExpiresIn: int64(s.tokens.AccessTTL() / time.Second),
	}, nil
}

func (s *Service) Logout(
	ctx context.Context,
	principal Principal,
	rawRefresh string,
) error {
	tenantID, hash, err := ParseRefreshToken(rawRefresh)
	if err != nil || tenantID != principal.TenantID {
		return nil
	}
	if err := s.repository.RevokeSession(ctx, principal.TenantID, principal.UserID, hash); err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, principal Principal) (CurrentUser, error) {
	user, err := s.repository.CurrentUser(ctx, principal.TenantID, principal.UserID)
	if errors.Is(err, ErrNotFound) {
		return CurrentUser{}, ErrUnauthorized
	}
	if err != nil {
		return CurrentUser{}, fmt.Errorf("load current user: %w", err)
	}
	return user.Current(), nil
}
