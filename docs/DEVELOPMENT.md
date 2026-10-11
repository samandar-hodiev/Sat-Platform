# Development setup

Commands below run from the repository root (`~/Desktop/SAT PLATFORM`). The path contains a
space, so quote it in shell commands.

## Prerequisites

| Tool | Version | Check |
| --- | --- | --- |
| Go | 1.26+ | `go version` |
| Node.js / npm | Node ≥ 22, npm 10 | `node -v && npm -v` |
| PostgreSQL | 15+ (developed on 18, Postgres.app) | `pg_isready -h localhost` |

Docker is not required.

## First time

```bash
make setup
```

This creates `.env` (API) and `web/.env.local` (web) from the examples and installs Go and npm
dependencies. Then create the database and point `.env` at it:

```bash
createdb -h localhost sat_platform
```

```dotenv
# .env
DATABASE_URL=postgres://<your-mac-user>@localhost:5432/sat_platform?sslmode=disable
```

Optional, for Telegram notifications after each push (needs GitPulse on localhost:8080):

```bash
make gitpulse-hook
```

## Run

Two terminals:

```bash
make api    # Go API  → http://localhost:8090
```

```bash
make web    # Next.js → http://localhost:3000
```

Check:

```bash
curl -s localhost:8090/health                   # 200 while the process runs
curl -s localhost:8090/ready                    # 200 when PostgreSQL answers, 503 otherwise
curl -s localhost:8090/api/v1/openapi.yaml | head
```

After changing Go code, restart `make api`. The web app hot-reloads.

## Environment files

| File | Read by | Contents |
| --- | --- | --- |
| `.env` (from `.env.example`) | Go API, development/test only | Database, CORS, limits, storage, later OAuth/AI secrets |
| `web/.env.local` (from `web/.env.example`) | Next.js | `NEXT_PUBLIC_*` only — bundled into browser JavaScript |

- Both are git-ignored. Never commit real values.
- Process environment variables override `.env`.
- The API validates its configuration at startup and lists every problem at once, e.g.
  `invalid configuration: DATABASE_URL is required`.
- `APP_ENV=production` reads no `.env` file and additionally requires `https://` origins and
  JSON logs.

## Tests and checks

```bash
make test          # Go unit tests + web unit tests (vitest)
make lint          # gofmt check, go vet, eslint, prettier check
make fmt           # format Go and web code
make typecheck     # TypeScript
make build         # api/bin/server + production web build
```

One Go package: `cd api && go test ./internal/<package>/...`.

Integration tests use a **disposable** database (never the development one):

```bash
createdb -h localhost sat_platform_test
make test-api-integration TEST_DATABASE_URL="postgres://<your-mac-user>@localhost:5432/sat_platform_test?sslmode=disable"
```

Without `TEST_DATABASE_URL` the integration tests are skipped, not failed.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `server: invalid configuration: …` | Fix the listed variables in `.env` |
| `/ready` returns 503 | `pg_isready -h localhost`; check `DATABASE_URL` |
| Port in use | `lsof -iTCP:8090 -sTCP:LISTEN` (or `3000`) and stop that process. 8080 belongs to GitPulse. |
| Browser CORS error | Add the exact web origin to `CORS_ALLOWED_ORIGINS`, restart the API |
| `NEXT_PUBLIC_API_URL is not set` | `cp web/.env.example web/.env.local` |
