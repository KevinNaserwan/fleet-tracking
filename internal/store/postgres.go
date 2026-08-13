package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KevinNaserwan/fleet-tracking/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (s *Postgres) FindForLogin(ctx context.Context, tenantSlug, email string) (auth.User, error) {
	const query = `
SELECT
    u.id::text, u.email::text, u.password_hash, u.full_name,
    t.id::text, t.slug::text, t.name, t.timezone
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE t.slug = $1
  AND t.is_active
  AND u.tenant_id = t.id
  AND u.email = $2
  AND u.status = 'active'
  AND u.deleted_at IS NULL
LIMIT 1`
	var user auth.User
	err := s.pool.QueryRow(ctx, query, tenantSlug, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Tenant.ID, &user.Tenant.Slug, &user.Tenant.Name, &user.Tenant.Timezone,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.User{}, err
	}
	if err := loadAuthorization(ctx, s.pool, user.Tenant.ID, user.ID, &user); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (s *Postgres) CreateSession(ctx context.Context, session auth.NewSession) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insert = `
INSERT INTO refresh_tokens (
    id, tenant_id, user_id, family_id, token_hash, expires_at, created_by_ip, user_agent
) VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6,
    NULLIF($7, '')::inet, NULLIF($8, '')
)`
	if _, err := tx.Exec(
		ctx, insert, session.ID, session.TenantID, session.UserID, session.FamilyID,
		session.TokenHash, session.ExpiresAt, session.Client.IP, session.Client.UserAgent,
	); err != nil {
		return err
	}
	const touch = `
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP
WHERE tenant_id = $1::uuid AND id = $2::uuid
  AND status = 'active' AND deleted_at IS NULL`
	tag, err := tx.Exec(ctx, touch, session.TenantID, session.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return auth.ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Postgres) RotateSession(
	ctx context.Context,
	tenantID string,
	currentHash []byte,
	next auth.RotationInput,
) (auth.User, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return auth.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockToken = `
SELECT id::text, user_id::text, family_id::text,
       expires_at > CURRENT_TIMESTAMP AS is_active, revoked_at
FROM refresh_tokens
WHERE tenant_id = $1::uuid AND token_hash = $2
FOR UPDATE`
	var tokenID, userID, familyID string
	var active bool
	var revokedAt *time.Time
	err = tx.QueryRow(ctx, lockToken, tenantID, currentHash).Scan(
		&tokenID, &userID, &familyID, &active, &revokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.User{}, err
	}
	if revokedAt != nil {
		if err := revokeFamily(ctx, tx, tenantID, familyID); err != nil {
			return auth.User{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return auth.User{}, err
		}
		return auth.User{}, auth.ErrRefreshReused
	}
	if !active {
		return auth.User{}, auth.ErrNotFound
	}

	const revokeCurrent = `
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE tenant_id = $1::uuid AND id = $2::uuid AND revoked_at IS NULL`
	tag, err := tx.Exec(ctx, revokeCurrent, tenantID, tokenID)
	if err != nil {
		return auth.User{}, err
	}
	if tag.RowsAffected() != 1 {
		return auth.User{}, auth.ErrRefreshReused
	}

	user, err := findCurrentUser(ctx, tx, tenantID, userID)
	if errors.Is(err, auth.ErrNotFound) {
		if familyErr := revokeFamily(ctx, tx, tenantID, familyID); familyErr != nil {
			return auth.User{}, familyErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return auth.User{}, commitErr
		}
		return auth.User{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.User{}, err
	}
	if err := loadAuthorization(ctx, tx, tenantID, userID, &user); err != nil {
		return auth.User{}, err
	}

	const insert = `
INSERT INTO refresh_tokens (
    id, tenant_id, user_id, family_id, token_hash, expires_at, created_by_ip, user_agent
) VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6,
    NULLIF($7, '')::inet, NULLIF($8, '')
)`
	if _, err := tx.Exec(
		ctx, insert, next.ID, tenantID, userID, familyID, next.TokenHash, next.ExpiresAt,
		next.Client.IP, next.Client.UserAgent,
	); err != nil {
		return auth.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (s *Postgres) RevokeSession(
	ctx context.Context,
	tenantID, userID string,
	hash []byte,
) error {
	const query = `
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP)
WHERE tenant_id = $1::uuid AND user_id = $2::uuid
  AND token_hash = $3 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, query, tenantID, userID, hash)
	return err
}

func (s *Postgres) CurrentUser(ctx context.Context, tenantID, userID string) (auth.User, error) {
	user, err := findCurrentUser(ctx, s.pool, tenantID, userID)
	if err != nil {
		return auth.User{}, err
	}
	if err := loadAuthorization(ctx, s.pool, tenantID, userID, &user); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type rowQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findCurrentUser(
	ctx context.Context,
	database rowQueryer,
	tenantID, userID string,
) (auth.User, error) {
	const query = `
SELECT
    u.id::text, u.email::text, u.full_name,
    t.id::text, t.slug::text, t.name, t.timezone
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.tenant_id = $1::uuid AND u.id = $2::uuid
  AND t.id = $1::uuid AND t.is_active
  AND u.status = 'active' AND u.deleted_at IS NULL
LIMIT 1`
	var user auth.User
	err := database.QueryRow(ctx, query, tenantID, userID).Scan(
		&user.ID, &user.Email, &user.FullName,
		&user.Tenant.ID, &user.Tenant.Slug, &user.Tenant.Name, &user.Tenant.Timezone,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrNotFound
	}
	return user, err
}

func loadAuthorization(
	ctx context.Context,
	database queryer,
	tenantID, userID string,
	user *auth.User,
) error {
	const query = `
SELECT DISTINCT r.name::text, p.permission_key::text
FROM user_roles ur
JOIN roles r
  ON r.tenant_id = ur.tenant_id AND r.id = ur.role_id
LEFT JOIN role_permissions rp
  ON rp.tenant_id = r.tenant_id AND rp.role_id = r.id
LEFT JOIN permissions p
  ON p.tenant_id = rp.tenant_id AND p.id = rp.permission_id
WHERE ur.tenant_id = $1::uuid AND ur.user_id = $2::uuid
ORDER BY r.name::text, p.permission_key::text`
	rows, err := database.Query(ctx, query, tenantID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	user.Roles = []string{}
	user.Permissions = []string{}
	roleSeen := make(map[string]struct{})
	permissionSeen := make(map[string]struct{})
	for rows.Next() {
		var role string
		var permission *string
		if err := rows.Scan(&role, &permission); err != nil {
			return err
		}
		if _, exists := roleSeen[role]; !exists {
			roleSeen[role] = struct{}{}
			user.Roles = append(user.Roles, role)
		}
		if permission != nil {
			if _, exists := permissionSeen[*permission]; !exists {
				permissionSeen[*permission] = struct{}{}
				user.Permissions = append(user.Permissions, *permission)
			}
		}
	}
	return rows.Err()
}

func revokeFamily(ctx context.Context, tx pgx.Tx, tenantID, familyID string) error {
	const query = `
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP)
WHERE tenant_id = $1::uuid AND family_id = $2::uuid AND revoked_at IS NULL`
	_, err := tx.Exec(ctx, query, tenantID, familyID)
	if err != nil {
		return fmt.Errorf("revoke refresh family: %w", err)
	}
	return nil
}
