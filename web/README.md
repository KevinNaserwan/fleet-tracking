# Fleet Tracking Web

Next.js 16 dashboard and backend-for-frontend for Fleet Tracking Milestone 1.

## Scope and data boundaries

- `/api/session/login` translates `tenant_slug` to the upstream `tenantSlug`, validates the documented response envelope, and stores access and refresh tokens in HttpOnly cookies.
- `/api/session` reads `/api/v1/me`. It performs one refresh attempt after an upstream `401`, rotates both cookies, and retries `/me` once.
- `/api/session/logout` attempts upstream revocation and always clears the local cookies after the attempt.
- Browser JavaScript never receives either token.
- Mutation routes require a same-origin `Origin` header and JSON content type.
- Dashboard identity, tenant, roles, and permissions are real `/me` data.
- The eight fleet rows, operational canvas, and activity timeline are explicitly labeled static demonstrations. Fleet data is planned for M2 and live tracking for M3.
- The infrastructure rail does not poll health endpoints or infer live dependency state.

## Environment

The Compose service provides these variables:

| Variable | Purpose | Local default |
| --- | --- | --- |
| `BACKEND_URL` | Server-only API origin | `http://api:8080` |
| `NEXT_PUBLIC_APP_URL` | Canonical browser origin used for mutation origin checks | `http://localhost:3000` |
| `SESSION_COOKIE_SECURE` | Set `true` when served over HTTPS | `false` |
| `PORT` | Next standalone server port | `3000` |
| `HOSTNAME` | Bind address | `0.0.0.0` |

Production must set `NEXT_PUBLIC_APP_URL` to the exact HTTPS origin and `SESSION_COOKIE_SECURE=true`. Do not place an API credential in a `NEXT_PUBLIC_*` variable.

## Local development

```bash
npm ci
npm run dev
```

The app expects the API at `BACKEND_URL`. The default Compose stack exposes the web service on `http://localhost:3000`.

## Container

The Dockerfile produces a Next standalone image and runs as the unprivileged `nextjs` user. Compose supplies a read-only root filesystem and a writable `/tmp` mount.

## Design system

- Direction: Indonesian fleet control room, light graphite and porcelain surfaces, safety-orange as the only accent.
- `DESIGN_VARIANCE: 4` for a stable operational grid with limited asymmetry.
- `MOTION_INTENSITY: 3` for direct state feedback only.
- `VISUAL_DENSITY: 8` for compact, scan-friendly data.
- Geometry: one compact 8px radius system.
- Typography: Geist Sans with Geist Mono for identifiers and operational labels.
- Icons: Phosphor only.
