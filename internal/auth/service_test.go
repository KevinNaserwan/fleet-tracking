package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type reuseRepository struct{}

func (reuseRepository) FindForLogin(context.Context, string, string) (User, error) {
	return User{}, ErrNotFound
}
func (reuseRepository) CreateSession(context.Context, NewSession) error { return nil }
func (reuseRepository) RotateSession(
	context.Context, string, []byte, RotationInput,
) (User, error) {
	return User{}, ErrRefreshReused
}
func (reuseRepository) RevokeSession(context.Context, string, string, []byte) error {
	return nil
}
func (reuseRepository) CurrentUser(context.Context, string, string) (User, error) {
	return User{}, ErrNotFound
}

func TestRefreshReuseIsUnauthorized(t *testing.T) {
	manager, err := NewTokenManager(
		"fleet-tracking",
		"fleet-dashboard",
		"0123456789abcdef0123456789abcdef",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	raw, _, err := NewRefreshToken(uuid.NewString())
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	service := NewService(reuseRepository{}, manager, 24*time.Hour)
	_, err = service.Refresh(context.Background(), raw, ClientMetadata{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh() error = %v, want ErrUnauthorized", err)
	}
}
