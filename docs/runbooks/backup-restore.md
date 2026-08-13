# PostgreSQL backup and restore runbook

This is the demo-environment procedure. Target RPO is 24 hours and target RTO is 4 hours. Production targets require an explicit business decision.

## Backup

1. Confirm the destination is outside the database volume and has sufficient encrypted storage.
2. Run a custom-format backup against a read-consistent database connection.
3. Record the database server version, migration version, timestamp, byte size, and SHA-256 checksum.
4. Copy the backup and checksum to the separate retention location.

```bash
mkdir -p backups
docker compose exec -T postgres sh -ceu '
  pg_dump \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    --format=custom \
    --no-owner \
    --no-acl
' > "backups/fleet-$(date -u +%Y%m%dT%H%M%SZ).dump"
sha256sum backups/fleet-*.dump > backups/SHA256SUMS
```

## Restore drill

Never test a restore over the active database.

1. Create an isolated PostgreSQL/PostGIS instance with the same major version.
2. Verify the checksum before opening the archive.
3. Restore into a new empty database with `pg_restore --clean --if-exists` only in that isolated target.
4. Run migration status, row-count sanity checks, tenant isolation probes, and application readiness.
5. Record duration and any manual step. Delete the drill environment after evidence is retained.

## Failure handling

If checksum validation, extension creation, schema restore, or readiness fails, do not promote the restored database. Preserve logs without secrets and escalate with the backup identifier and failing step.
