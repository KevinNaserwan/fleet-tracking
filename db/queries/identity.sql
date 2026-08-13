-- name: GetTenantBySlug :one
SELECT id, name, slug, timezone, is_active, created_at, updated_at
FROM tenants
WHERE slug = sqlc.arg(slug)
LIMIT 1;

-- name: GetUserForLogin :one
SELECT
    u.id,
    u.tenant_id,
    u.email,
    u.password_hash,
    u.full_name,
    u.status,
    u.last_login_at,
    u.created_at,
    u.updated_at
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE t.slug = sqlc.arg(tenant_slug)
  AND t.is_active
  AND u.email = sqlc.arg(email)
  AND u.status = 'active'
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: GetUserByTenantAndID :one
SELECT id, tenant_id, email, full_name, status, last_login_at, created_at, updated_at
FROM users
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(user_id)
  AND deleted_at IS NULL
LIMIT 1;

-- name: TouchUserLastLogin :exec
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP
WHERE tenant_id = sqlc.arg(tenant_id)
  AND id = sqlc.arg(user_id)
  AND deleted_at IS NULL;

-- name: ListUserAuthorization :many
SELECT DISTINCT
    r.id AS role_id,
    r.name AS role_name,
    p.permission_key
FROM user_roles ur
JOIN roles r
  ON r.tenant_id = ur.tenant_id
 AND r.id = ur.role_id
LEFT JOIN role_permissions rp
  ON rp.tenant_id = r.tenant_id
 AND rp.role_id = r.id
LEFT JOIN permissions p
  ON p.tenant_id = rp.tenant_id
 AND p.id = rp.permission_id
WHERE ur.tenant_id = sqlc.arg(tenant_id)
  AND ur.user_id = sqlc.arg(user_id)
ORDER BY r.name, p.permission_key;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    id, tenant_id, user_id, family_id, token_hash, expires_at, created_by_ip, user_agent
) VALUES (
    sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(user_id), sqlc.arg(family_id),
    sqlc.arg(token_hash), sqlc.arg(expires_at), sqlc.narg(created_by_ip), sqlc.narg(user_agent)
)
RETURNING id, tenant_id, user_id, family_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetActiveRefreshTokenByHash :one
SELECT id, tenant_id, user_id, family_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- name: GetRefreshTokenByHash :one
SELECT id, tenant_id, user_id, family_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = sqlc.arg(token_hash)
LIMIT 1;

-- name: RevokeRefreshTokenByHash :execrows
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP)
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL;

-- name: RevokeRefreshTokenFamily :execrows
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP)
WHERE tenant_id = sqlc.arg(tenant_id)
  AND family_id = sqlc.arg(family_id)
  AND revoked_at IS NULL;
