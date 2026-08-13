package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrRefreshReused = errors.New("refresh token reuse detected")
)

type Tenant struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

type User struct {
	ID           string
	Tenant       Tenant
	Email        string
	PasswordHash string
	FullName     string
	Roles        []string
	Permissions  []string
}

type CurrentUser struct {
	ID          string   `json:"id"`
	Tenant      Tenant   `json:"tenant"`
	Email       string   `json:"email"`
	FullName    string   `json:"fullName"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func (u User) Current() CurrentUser {
	roles := append([]string(nil), u.Roles...)
	permissions := append([]string(nil), u.Permissions...)
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return CurrentUser{
		ID:          u.ID,
		Tenant:      u.Tenant,
		Email:       u.Email,
		FullName:    u.FullName,
		Roles:       roles,
		Permissions: permissions,
	}
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type Principal struct {
	TenantID   string
	UserID     string
	Permissions []string
	TokenID    string
}

type LoginInput struct {
	TenantSlug string
	Email      string
	Password   string
}

type ClientMetadata struct {
	IP        string
	UserAgent string
}

type NewSession struct {
	ID        string
	TenantID  string
	UserID    string
	FamilyID  string
	TokenHash []byte
	ExpiresAt time.Time
	Client    ClientMetadata
}

type RotationInput struct {
	ID        string
	TokenHash []byte
	ExpiresAt time.Time
	Client    ClientMetadata
}

type Repository interface {
	FindForLogin(context.Context, string, string) (User, error)
	CreateSession(context.Context, NewSession) error
	RotateSession(context.Context, string, []byte, RotationInput) (User, error)
	RevokeSession(context.Context, string, string, []byte) error
	CurrentUser(context.Context, string, string) (User, error)
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
