# ADR 0001: Modular monolith with separate process roles

- Status: Accepted
- Date: 2026-08-13

## Context

The MVP spans identity, fleet, tracking, spatial rules, trips, alerts, and notifications. A solo build needs strong boundaries without the deployment and contract burden of full microservices.

## Decision

Keep domain modules in one Go module and one repository. Deploy API, worker, outbox publisher, and simulator as separate binaries from the same source tree.

## Consequences

- Cross-module transactions remain straightforward.
- Refactoring boundaries is cheaper while the domain is still changing.
- Worker workloads can scale independently.
- Module discipline must be enforced in code review because the compiler does not prevent every cross-module shortcut.
- A module may later be extracted only after volume, ownership, or reliability requirements justify it.
