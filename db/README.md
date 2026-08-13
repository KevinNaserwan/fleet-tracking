# Milestone 1 database

Apply `migrations/000001_milestone1.up.sql` to an empty PostgreSQL database with permission to install PostGIS, `citext`, and `pgcrypto`. The matching down migration removes the Milestone 1 application objects but intentionally leaves extensions in place because they may have existed before the application migration.

`seeds/001_demo.sql` is rerunnable development data. It creates tenant `nusantara-logistics` and eight Jakarta-area fleet vehicles.

Development login only:

- Tenant: `nusantara-logistics`
- Email: `admin@fleet.local`
- Password: `Admin123!`

The seed persists only a valid Argon2id hash for the password. Refresh-token values are never persisted; application code must store `SHA-256(raw_refresh_token)` as the 32-byte `refresh_tokens.token_hash` value.
