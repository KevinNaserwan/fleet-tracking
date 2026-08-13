package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	manager, err := NewTokenManager(
		"fleet-tracking",
		"fleet-dashboard",
		"0123456789abcdef0123456789abcdef",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	user := User{
		ID: uuid.NewString(),
		Tenant: Tenant{ID: uuid.NewString()},
		Permissions: []string{"fleet:read"},
	}
	raw, err := manager.Sign(user)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	principal, err := manager.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if principal.TenantID != user.Tenant.ID || principal.UserID != user.ID ||
		len(principal.Permissions) != 1 || principal.Permissions[0] != "fleet:read" {
		t.Fatalf("Parse() principal = %#v", principal)
	}
}

func TestOpaqueRefreshTokenRoundTrip(t *testing.T) {
	tenantID := uuid.NewString()
	raw, createdHash, err := NewRefreshToken(tenantID)
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	parsedTenantID, parsedHash, err := ParseRefreshToken(raw)
	if err != nil {
		t.Fatalf("ParseRefreshToken() error = %v", err)
	}
	if parsedTenantID != tenantID || string(createdHash) != string(parsedHash) {
		t.Fatal("refresh token tenant or hash did not round trip")
	}
}
