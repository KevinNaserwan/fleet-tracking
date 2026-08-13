# ADR 0002: Transactional outbox and at-least-once delivery

- Status: Accepted
- Date: 2026-08-13

## Context

Writing telemetry to PostgreSQL and publishing directly to RabbitMQ creates a failure window between two independent systems.

## Decision

Write the domain mutation and an `outbox_events` row in one PostgreSQL transaction. A publisher leases unpublished rows, publishes durable messages with confirms, and marks successful rows. Consumers store a `(consumer_name, message_id)` receipt in the same transaction as their effect.

## Consequences

- A committed event is eventually publishable after broker recovery.
- Duplicate publication remains possible, so consumer idempotency is mandatory.
- Publication lag and failed attempts require metrics and operational alerts.
- Ordering is defined only where an aggregate sequence is explicitly introduced; global order is not promised.
