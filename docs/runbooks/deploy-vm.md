# Production-like VM deployment

This runbook deploys the Milestone 1 demo behind the host nginx. It does not claim the availability or load targets reserved for later milestone evidence.

## Application deployment

Use a secret file that is not committed:

```bash
docker compose \
  --env-file .env.production \
  -f docker-compose.yml \
  -f deployments/docker-compose.production.yml \
  --profile tools run --rm migrate

docker compose \
  --env-file .env.production \
  -f docker-compose.yml \
  -f deployments/docker-compose.production.yml \
  --profile tools run --rm seed

docker compose \
  --env-file .env.production \
  -f docker-compose.yml \
  -f deployments/docker-compose.production.yml \
  up -d --no-build api worker outbox-publisher web prometheus grafana otel-collector
```

The production override publishes only these loopback ports:

| Service | Default loopback port |
|---|---:|
| API | `18180` |
| Web | `13100` |
| Prometheus | `19090` |
| Grafana | `13001` |

PostgreSQL, Redis, RabbitMQ, and RabbitMQ management are not published to the host.

## Host edge activation

These commands require host root access because nginx and Let's Encrypt state are root-owned:

```bash
sudo install -m 0644 deployments/nginx/fleet.meetsin.id.http.conf \
  /etc/nginx/sites-available/fleet.meetsin.id
sudo ln -sfn /etc/nginx/sites-available/fleet.meetsin.id \
  /etc/nginx/sites-enabled/fleet.meetsin.id
sudo nginx -t
sudo systemctl reload nginx
sudo certbot --nginx -d fleet.meetsin.id --redirect
sudo nginx -t
```

Certbot owns the HTTPS certificate paths and renewal timer after activation. Never reuse another hostname's certificate.

## Smoke gate

```bash
curl --fail --silent https://fleet.meetsin.id/health/live
curl --fail --silent https://fleet.meetsin.id/health/ready
curl --fail --silent --head https://fleet.meetsin.id/login
```

Validate login through the browser and confirm access/refresh tokens are absent from browser JavaScript storage. The BFF cookies must be `HttpOnly`, `Secure`, and `SameSite=Lax`.

## Rollback

Keep the previous image tag in `.env.production`. Roll back applications without rolling back schema unless the migration has separately passed its down-migration review:

```bash
IMAGE_TAG=<previous-tag> docker compose \
  --env-file .env.production \
  -f docker-compose.yml \
  -f deployments/docker-compose.production.yml \
  up -d --no-build api worker outbox-publisher web
```
