# Milestone 1: Foundation

## Scope

- Repository and modular Go process structure
- Local PostGIS, Redis, RabbitMQ, Prometheus, Grafana, and OTel stack
- Strict configuration and graceful process lifecycle
- Versioned migration and rerunnable demo seed
- Login, refresh rotation, logout, current profile, permission claims, and tenant context
- Health, metrics, structured logs, standard errors, and OpenAPI skeleton
- Secure dashboard BFF and operations console shell
- CI build, migration round trip, tests, and security scanning

## Acceptance gate

| Check | Evidence required |
|---|---|
| Config rejects missing secrets | Unit test or startup failure with redacted message |
| Migration is reversible | CI up, down-all, up job |
| Admin can login | API smoke response and `/me` permission claims |
| Refresh rotates | Old token rejected and family reuse test |
| Tenant boundary exists | Repository signatures and cross-tenant test scaffold |
| Dependencies are private | Compose publishes data ports on loopback only |
| Processes shut down cleanly | Signal-handling test or observed bounded shutdown |
| UI handles session states | Login, loading, authenticated, expired, and error paths |

The gate is not considered passed merely because files exist. Validation results belong in the release or pull request evidence.

## Next milestone, not implemented here

Milestone 2 adds vehicle/device CRUD, assignment history, device credentials, telemetry ingestion, idempotency, monotonic latest position, the full transactional outbox publisher, and a configurable route simulator.
