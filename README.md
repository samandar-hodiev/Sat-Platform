# SAT Platform

Self-study preparation for the Digital SAT: diagnostic → personal study plan → lessons and
practice → mistake review → real-format mock exams → progress → AI assistant. Learners use the
web app (Android and iOS apps follow, built with Flutter); the Owner runs content, learners and
the platform from the Owner dashboard.

**One Go API, one REST contract (`/api/v1`) and one PostgreSQL database serve every client.**
Business rules live in the API. Product requirements: [docs/PRD.md](docs/PRD.md).

| Part | Stack |
| --- | --- |
| Web (learner + Owner) | Next.js 16 (App Router), React 19, TypeScript (strict), Tailwind CSS 4, ESLint, Prettier, Vitest |
| API | Go 1.26, Gin, pgx, log/slog, OpenAPI 3.1 |
| Database | PostgreSQL 15+ |
| Mobile (planned) | Flutter — Android and iOS from one codebase |

> Status: **foundation only** (session 0.00). No business features, pages or tables yet; they
> are built by the sessions listed in [CLAUDE.md](CLAUDE.md). Progress: [docs/PROGRESS.md](docs/PROGRESS.md).

## Repository

```text
api/        Go REST API — cmd/server, internal/{config,server,middleware,health,httpx,apperr,platform}
api/openapi OpenAPI contract, served at /api/v1/openapi.yaml
web/        Next.js app (learner app + Owner dashboard)
mobile/     Flutter app plan (README only)
docs/       PRD, architecture, API, database, authentication, development, progress
infrastructure/scripts  GitPulse pre-push hook (Telegram push notifications)
```

Details: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Requirements

Go 1.26+, Node.js 22+ with npm 10, PostgreSQL 15+. Docker is not needed.

## Getting started

```bash
make setup                                   # .env + web/.env.local from examples, Go + npm deps
createdb -h localhost sat_platform           # then set DATABASE_URL in .env
make api                                     # http://localhost:8090
make web                                     # http://localhost:3000 (second terminal)
curl -s localhost:8090/ready                 # {"success":true,"data":{"status":"ok",…}}
```

Full guide, environment variables and troubleshooting: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Configuration

- `.env` (from [`.env.example`](.env.example)) configures the Go API: `APP_ENV`, `API_PORT`,
  `WEB_URL`, `DATABASE_URL` (required), `CORS_ALLOWED_ORIGINS`, rate limit, storage.
- `web/.env.local` (from [`web/.env.example`](web/.env.example)) holds public values only:
  `NEXT_PUBLIC_API_URL`.
- Both files are git-ignored. Secrets (database password, OAuth secrets, AI keys) never go into
  the web app or the repository.

## Tests and checks

```bash
make test                     # Go + web unit tests
make test-api-integration TEST_DATABASE_URL="postgres://<user>@localhost:5432/sat_platform_test?sslmode=disable"
make lint                     # gofmt check, go vet, eslint, prettier
make typecheck
make build
```

## Documentation

| | |
| --- | --- |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System, one backend for web + mobile, module boundaries, decisions |
| [API.md](docs/API.md) | Conventions (envelope, errors, pagination, dates, versioning) + endpoint list |
| [AUTHENTICATION.md](docs/AUTHENTICATION.md) | Web and mobile auth, tokens, refresh, logout, Google |
| [DB.md](docs/DB.md) | Database rules, migration strategy, naming, integrity |
| [DEVELOPMENT.md](docs/DEVELOPMENT.md) | Local setup, commands, tests |
| [mobile/README.md](mobile/README.md) | Flutter plan |

## Push notifications

`make gitpulse-hook` installs a pre-push hook that announces each pushed commit in Telegram
through the local GitPulse service (`localhost:8080`). A push never fails because of it.
