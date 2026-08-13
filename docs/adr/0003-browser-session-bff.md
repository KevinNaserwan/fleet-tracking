# ADR 0003: Browser session through a Next.js BFF

- Status: Accepted
- Date: 2026-08-13

## Context

Persisting access or refresh tokens in browser storage increases exposure during an XSS event. The dashboard and API are deployed together for the portfolio MVP.

## Decision

The browser authenticates through Next.js route handlers. The BFF stores tokens in secure `HttpOnly` cookies and forwards the short-lived access token server-side. The Go API remains usable by non-browser clients through its documented bearer contract.

## Consequences

- Browser JavaScript cannot read refresh tokens.
- Same-site deployment is the preferred production topology.
- State-changing BFF endpoints must retain SameSite, Origin, and CSRF protections.
- Native or third-party clients can use the API contract without the BFF.
