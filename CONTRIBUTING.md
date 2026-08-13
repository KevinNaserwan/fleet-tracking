# Contributing

## Workflow

1. Create a focused branch from `main`.
2. Keep module boundaries explicit and put interfaces on the consumer side only when substitution is needed.
3. Add or update migration, tests, OpenAPI, metrics, and error behavior with each relevant change.
4. Run `make verify` before requesting review.
5. Use Conventional Commits, for example `feat(identity): rotate refresh token families`.

## Engineering rules

- A tenant-scoped query without `tenant_id` is a release blocker.
- Do not publish an event outside the database transaction; write an outbox record instead.
- Consumers must be idempotent because broker delivery is at least once.
- Do not put high-cardinality identifiers into Prometheus labels.
- Do not introduce an abstraction before a real second implementation or test seam needs it.
- Do not put secrets or personal data into fixtures, logs, traces, screenshots, or commits.

## Database changes

Migration filenames are sequential and immutable after merge. Every schema change needs a safe forward migration and a tested rollback where data semantics permit one.
