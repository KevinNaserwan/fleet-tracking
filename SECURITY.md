# Security policy

## Supported version

Only the latest commit on `main` is supported during MVP development.

## Reporting a vulnerability

Do not open a public issue for suspected vulnerabilities. Send a private report to the repository owner with:

- affected endpoint, component, and commit;
- a minimal reproduction without real customer data;
- expected and observed impact;
- suggested mitigation when available.

The project targets acknowledgement within 72 hours. No remediation SLA is promised for this portfolio environment.

## Security invariants

- Every business row is scoped by `tenant_id` at repository and service boundaries.
- Raw refresh tokens, passwords, API keys, webhook secrets, and authorization headers are never stored or logged.
- Access tokens are short-lived; refresh tokens are hashed, rotated, revocable, and protected by family reuse detection.
- Public deployments terminate TLS before application traffic.
- PostgreSQL, Redis, RabbitMQ, Prometheus, Grafana, and `/metrics` remain private.
- SQL uses bound parameters. CORS is an explicit allowlist.
- Containers run without root privileges where the upstream image supports it.

Development credentials in seed data are public and must never be reused outside local/demo environments.
