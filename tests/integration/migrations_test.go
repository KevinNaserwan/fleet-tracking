//go:build integration

package integration_test

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/argon2"
)

const demoPasswordHash = "$argon2id$v=19$m=65536,t=3,p=4$ZmxlZXQtZGV2LXNlZWQhIQ$C2RLxCAz4oX2OJaJ9Se+7xjRYx7nnQm42JULOKOaa3w"

func TestMilestoneOneMigrationAndSeed(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required; use a disposable empty PostgreSQL/PostGIS database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	up := readRepoFile(t, "db", "migrations", "000001_milestone1.up.sql")
	seed := readRepoFile(t, "db", "seeds", "001_demo.sql")
	down := readRepoFile(t, "db", "migrations", "000001_milestone1.down.sql")

	if _, err := db.ExecContext(ctx, up); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}
	cleanedUp := false
	defer func() {
		if !cleanedUp {
			_, _ = db.ExecContext(context.Background(), down)
		}
	}()

	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("apply first seed pass: %v", err)
	}
	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("apply second seed pass: %v", err)
	}

	assertCount(t, ctx, db, 1, `SELECT count(*) FROM tenants WHERE slug = 'nusantara-logistics'`)
	assertCount(t, ctx, db, 1, `
		SELECT count(*) FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE t.slug = 'nusantara-logistics' AND u.email = 'admin@fleet.local'`)
	assertCount(t, ctx, db, 8, `
		SELECT count(*) FROM vehicles v
		JOIN tenants t ON t.id = v.tenant_id
		WHERE t.slug = 'nusantara-logistics'`)
	assertCount(t, ctx, db, 8, `
		SELECT count(*) FROM device_assignments da
		JOIN tenants t ON t.id = da.tenant_id
		WHERE t.slug = 'nusantara-logistics' AND da.ended_at IS NULL`)
	assertCount(t, ctx, db, 3, `
		SELECT count(*) FROM geofences g
		JOIN tenants t ON t.id = g.tenant_id
		WHERE t.slug = 'nusantara-logistics' AND g.geofence_type = 'circle'`)

	var passwordHash string
	if err := db.QueryRowContext(ctx, `
		SELECT u.password_hash
		FROM users u JOIN tenants t ON t.id = u.tenant_id
		WHERE t.slug = 'nusantara-logistics' AND u.email = 'admin@fleet.local'`).Scan(&passwordHash); err != nil {
		t.Fatalf("read demo password hash: %v", err)
	}
	if passwordHash != demoPasswordHash || !strings.HasPrefix(passwordHash, "$argon2id$v=19$") {
		t.Fatalf("demo password is not the documented Argon2id hash")
	}
	if !verifyArgon2idPassword(passwordHash, "Admin123!") {
		t.Fatal("demo Argon2id hash does not verify the documented password")
	}

	var validSpatialData bool
	if err := db.QueryRowContext(ctx, `
		SELECT bool_and(ST_SRID(position) = 4326 AND ST_GeometryType(position) = 'ST_Point')
		FROM vehicle_latest_positions`).Scan(&validSpatialData); err != nil {
		t.Fatalf("inspect latest-position geometries: %v", err)
	}
	if !validSpatialData {
		t.Fatal("latest positions must all be non-empty SRID 4326 points")
	}

	assertSecondActiveAssignmentRejected(t, ctx, db)

	if _, err := db.ExecContext(ctx, down); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
	cleanedUp = true

	var tenantsTable sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.tenants')::text`).Scan(&tenantsTable); err != nil {
		t.Fatalf("check down migration: %v", err)
	}
	if tenantsTable.Valid {
		t.Fatalf("down migration left table %q", tenantsTable.String)
	}
}

func verifyArgon2idPassword(encodedHash, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	var memory, iterations uint32
	var parallelism uint8
	if scanned, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil || scanned != 3 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func assertSecondActiveAssignmentRejected(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	_, insertErr := db.ExecContext(ctx, `
		INSERT INTO device_assignments (tenant_id, device_id, vehicle_id)
		SELECT da.tenant_id, da.device_id, v.id
		FROM device_assignments da
		JOIN vehicles v ON v.tenant_id = da.tenant_id AND v.id <> da.vehicle_id
		WHERE da.ended_at IS NULL
		LIMIT 1`)
	if insertErr == nil {
		t.Fatal("expected unique violation for a second active device assignment")
	}
}

func assertCount(t *testing.T, ctx context.Context, db *sql.DB, want int, query string) {
	t.Helper()

	var got int
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for query %q", got, want, query)
	}
}

func readRepoFile(t *testing.T, pathParts ...string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	content, err := os.ReadFile(filepath.Join(append([]string{repoRoot}, pathParts...)...))
	if err != nil {
		t.Fatal(errors.Join(fmt.Errorf("read repository SQL"), err))
	}
	return string(content)
}
