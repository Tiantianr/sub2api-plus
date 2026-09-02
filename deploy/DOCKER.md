# Sub2API Plus Docker Image

Sub2API Plus is an AI API Gateway Platform for distributing and managing AI product subscription API quotas.

## Quick Start

```bash
docker run -d \
  --name sub2api \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/sub2api" \
  -e REDIS_URL="redis://host:6379" \
  ghcr.io/tiantianr/sub2api-plus:vX.Y.Z-custom.NNN
```

## Docker Compose

```yaml
version: '3.8'

services:
  sub2api:
    image: ghcr.io/tiantianr/sub2api-plus:vX.Y.Z-custom.NNN
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@db:5432/sub2api?sslmode=disable
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=sub2api
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Yes | - |
| `REDIS_URL` | Redis connection string | Yes | - |
| `PORT` | Server port | No | `8080` |
| `GIN_MODE` | Gin framework mode (`debug`/`release`) | No | `release` |
| `WEBAUTHN_ENABLED` | Enable the WebAuthn deployment boundary | No | `false` |
| `WEBAUTHN_RP_DISPLAY_NAME` | Relying-party display name | No | `Sub2API` |
| `WEBAUTHN_RP_ID` | Relying-party domain without scheme or port | When enabled | - |
| `WEBAUTHN_RP_ORIGINS` | Comma-separated allowed origins | When enabled | - |

Passkey remains disabled until the WebAuthn values are valid and an
administrator explicitly enables it in System Settings. Recreate the
application container after changing these values.

## Supported Architectures

- `linux/arm64`

Personal releases from `v0.1.178+custom.902` onward target the current
production architecture only. Other operating systems and CPU architectures
are not published by this distribution.

## Tags

- `vX.Y.Z-custom.NNN` - Immutable fork release, for example `v0.1.183-custom.924`

Git and GitHub Releases use `vX.Y.Z+custom.NNN`. The release workflow
preserves the leading `v` and replaces only `+` with `-` to produce the
OCI-compatible image tag. For example:

```text
Git/GitHub: v0.1.183+custom.924
GHCR:       ghcr.io/tiantianr/sub2api-plus:v0.1.183-custom.924
```

After verification, record the published digest and pin
`ghcr.io/tiantianr/sub2api-plus:<version>@sha256:<digest>` in production. This
distribution does not move shared `latest`, major, or minor aliases.

## Links

- [GitHub Repository](https://github.com/Tiantianr/sub2api-plus)
- [Documentation](https://github.com/Tiantianr/sub2api-plus#readme)
