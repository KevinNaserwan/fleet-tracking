package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type AccessClaims struct {
	TenantID   string   `json:"tenant_id"`
	UserID     string   `json:"user_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	issuer   string
	audience string
	secret   []byte
	ttl      time.Duration
	now      func() time.Time
}

func NewTokenManager(issuer, audience, secret string, ttl time.Duration) (*TokenManager, error) {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" ||
		len([]byte(secret)) < 32 || ttl <= 0 {
		return nil, errors.New("invalid token manager configuration")
	}
	return &TokenManager{
		issuer: issuer, audience: audience, secret: []byte(secret), ttl: ttl, now: time.Now,
	}, nil
}

func (m *TokenManager) Sign(user User) (string, error) {
	now := m.now().UTC()
	claims := AccessClaims{
		TenantID:   user.Tenant.ID,
		UserID:     user.ID,
		Permissions: append([]string(nil), user.Permissions...),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) Parse(raw string) (Principal, error) {
	claims := new(AccessClaims)
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || token == nil || !token.Valid ||
		claims.Subject != claims.UserID || claims.ID == "" {
		return Principal{}, ErrInvalidToken
	}
	if _, err := uuid.Parse(claims.TenantID); err != nil {
		return Principal{}, ErrInvalidToken
	}
	if _, err := uuid.Parse(claims.UserID); err != nil {
		return Principal{}, ErrInvalidToken
	}
	return Principal{
		TenantID: claims.TenantID,
		UserID: claims.UserID,
		Permissions: append([]string(nil), claims.Permissions...),
		TokenID: claims.ID,
	}, nil
}

func (m *TokenManager) AccessTTL() time.Duration {
	return m.ttl
}

func NewRefreshToken(tenantID string) (string, []byte, error) {
	parsedTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("invalid tenant ID: %w", err)
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", nil, fmt.Errorf("generate refresh token: %w", err)
	}
	raw := "rt1." + parsedTenantID.String() + "." + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(raw))
	return raw, hash[:], nil
}

func ParseRefreshToken(raw string) (string, []byte, error) {
	if len(raw) < 32 || len(raw) > 2048 {
		return "", nil, ErrInvalidToken
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || parts[0] != "rt1" {
		return "", nil, ErrInvalidToken
	}
	tenantID, err := uuid.Parse(parts[1])
	if err != nil {
		return "", nil, ErrInvalidToken
	}
	random, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || len(random) != 32 {
		return "", nil, ErrInvalidToken
	}
	hash := sha256.Sum256([]byte(raw))
	return tenantID.String(), hash[:], nil
}
