# Architecture

SAT Self-Study Platform: learners prepare for the Digital SAT; the Owner runs content,
learners and the platform from the Owner dashboard. This document is the structure every
session follows (CLAUDE.md → "Aniq papka tuzilmasini 0.00 sessiyasi belgilaydi").

## 1. One backend for every client

```text
   Web app (Next.js)          Owner dashboard (Next.js, /owner)        Android + iOS (Flutter, later)
          \                              |                                     /
           \_____________ HTTPS · JSON · /api/v1 · same contract _____________/
                                         |
                       ┌─────────────────▼──────────────────┐
                       │  Go API — Gin, modular monolith    │──── AI provider (3.12, server-side keys)
                       │  auth · authorization · rules      │──── File storage (local → S3-compatible)
                       └─────────────────┬──────────────────┘──── Email provider (1.01)
                                         │
                                   PostgreSQL (single source of truth)
```

- **One Go API, one REST contract (`/api/v1`), one PostgreSQL database** for the learner web
  app, the Owner dashboard, Android and iOS. There is no mobile backend and no mobile database.
- **Business rules live in the API**: permissions, Premium feature gating, usage limits, exam
  timing and scoring, placement results, study-plan generation. Clients render what the API
  returns; hiding something in a client is cosmetic, never a security measure.
- **Clients are interchangeable**: no endpoint behaves differently per client. Authentication
  works without browser cookies for mobile ([AUTHENTICATION.md](AUTHENTICATION.md)).
- The machine-readable contract is `api/openapi/openapi.yaml` (served at
  `/api/v1/openapi.yaml`); the Flutter apps generate their Dart client from it.

## 2. Repository layout

```text
SAT PLATFORM/
├── api/                          Go API (module github.com/samandar-hodiev/sat-platform/api)
│   ├── cmd/server/main.go        entrypoint: config → logger → DB pool → router → graceful run
│   ├── internal/
│   │   ├── config/               env loading + validation (fails fast, lists every problem)
│   │   ├── server/               router assembly, middleware order, module registration, Run
│   │   ├── middleware/           request id, access log, recovery, security headers, CORS,
│   │   │                         body limit, error rendering, rate limit
│   │   ├── health/               GET /health (liveness), GET /ready (readiness)
│   │   ├── apperr/               client-safe error codes → HTTP status
│   │   ├── httpx/                response envelope, JSON/query binding + validation, pagination
│   │   ├── platform/
│   │   │   ├── database/         pgx connection pool
│   │   │   ├── logger/           slog (text in dev, JSON in prod)
│   │   │   ├── ratelimit/        Limiter interface + in-memory implementation
│   │   │   └── storage/          ObjectStorage interface + local implementation
│   │   └── <module>/             one folder per domain, added by its session (auth, users, …)
│   ├── openapi/                  openapi.yaml (embedded and served) + contract tests
│   └── migrations/               SQL migrations (tool and first tables: session 0.01)
├── web/                          Next.js app (learner app + Owner dashboard)
│   ├── src/app/                  App Router routes and layouts
│   ├── src/lib/api/              the only HTTP client (envelope unwrap, ApiError)
│   ├── src/lib/env.ts            public runtime config (NEXT_PUBLIC_* only)
│   └── src/types/api.ts          API contract types
├── mobile/README.md              Flutter app plan (not built yet)
├── docs/                         PRD, ARCHITECTURE, API, DB, AUTHENTICATION, DEVELOPMENT, PROGRESS
├── infrastructure/scripts/       GitPulse pre-push hook (Telegram notifications)
├── .env.example                  API configuration (root .env, git-ignored)
├── Makefile                      developer commands
└── CLAUDE.md                     session map and rules
```

Folders are created when there is code for them. Feature sessions add `api/internal/<module>/`,
`web/src/features/<feature>/`, `web/src/components/` (0.02 UI kit) and so on.

## 3. Backend

**Request lifecycle**

```text
request → RequestID → AccessLog → Recovery → SecurityHeaders → CORS → BodyLimit → Errors
        → /health, /ready                      (no rate limit: probes are polled constantly)
        → /api/v1 → RateLimit(per IP) → [Authenticate, RequireRole/Permission — 1.01/1.02]
        → handler → service → repository (pgx, parameterised SQL) → PostgreSQL
        ← { success, data | error, meta }
```

**Module shape** (`api/internal/<module>/`), e.g. `questions`:

| File | Responsibility |
| --- | --- |
| `module.go` | `NewModule(Deps)` and `RegisterRoutes(v1 *gin.RouterGroup)` |
| `handler.go` | HTTP only: bind + validate input (`httpx.BindJSON`), call service, write response |
| `service.go` | Business rules; returns `apperr` errors for client-facing failures |
| `repository.go` | SQL with `$1` parameters, transactions for multi-row writes |
| `*_test.go` | Unit tests for rules; integration tests skip without `TEST_DATABASE_URL` |

Rules:

- A module owns its tables. Modules never query another module's tables; the consumer
  declares a small interface and `internal/server` wires the implementation in.
- Handlers call `httpx.Fail(c, err)`; the `Errors` middleware renders `apperr.Error` as is and
  turns anything else into `INTERNAL_ERROR` (cause logged with the request id, never returned).
- Validation: `binding` tags on request structs; errors become `422 VALIDATION_ERROR` with
  `details.fields` keyed by JSON field name.
- Logging: `log/slog`, one access line per request (method, route template, status, latency,
  request id, client IP). Never log bodies, tokens, passwords, codes or full query strings.
- Configuration: only `internal/config` reads the environment. A new setting is added to
  `Config`, validated there, and documented in `.env.example` in the same change.
- Rate limiting: `/api/v1` has a per-IP default (`RATE_LIMIT_PER_MINUTE`). Sensitive routes
  add `middleware.RateLimit(limiter, "<scope>", n, window, keyFunc)` with their own budget —
  they reuse `ratelimit.Limiter`, never write their own. The in-memory implementation is per
  instance; a shared store replaces it before running more than one API instance (5.05).
- Storage: modules depend on `storage.ObjectStorage`; keys are generated by the module
  (`avatars/<user-id>/<uuid>.webp`), never taken from user file names. Upload validation (size,
  magic bytes, type allow-list) happens in the module before `Put`.
- Deployment unit: one binary (`cmd/server`). Background jobs, if needed later, become a
  second command in the same module, not a second service.

**Planned modules** (from the session map; each created by its session, none exist yet):
auth, owner auth/RBAC, users & profiles, preferences, onboarding/goals, study plans,
curriculum, lessons, questions (+ versions), practice, mock exams, exam results, placement,
bookmarks, mistakes, formula library, AI, subscriptions & payments, notifications,
analytics, platform settings, audit logs.

## 4. Frontend

- Next.js 16 App Router, React 19, TypeScript `strict` + `noUncheckedIndexedAccess`,
  Tailwind CSS v4, ESLint (`eslint-config-next`), Prettier, Vitest.
- **Route groups** (created by their sessions):

  | Group | Paths | Session |
  | --- | --- | --- |
  | `(public)` | `/`, `/pricing`, `/faq`, `/privacy`, `/terms`, `/contact` | 0.03 |
  | `(auth)` | `/auth/sign-in`, `/auth/sign-up`, `/auth/forgot-password`, … | 1.01 |
  | `(learner)` | `/app/*` — dashboard, plan, practice, exams, progress, settings | 2.xx, 3.xx |
  | `(focus)` | full-screen mock exam runner without the sidebar | 3.08 |
  | `owner/login` | Owner sign-in (separate layout) | 1.02 |
  | `owner/(dashboard)` | `/owner/*` — CMS, learners, analytics, settings, audit | 4.xx, 3.xx |

  Learner and Owner have separate layouts and navigation; access is enforced by the API.
- **API access**: only through `src/lib/api/client.ts`. Features add typed functions in
  `src/features/<feature>/api.ts`; components never call `fetch`. Contract types live in
  `src/types/api.ts` and follow `openapi.yaml`.
- **State conventions**: every data view handles loading, empty, error (with retry) and success.
  `app/loading.tsx`, `app/error.tsx` and `app/not-found.tsx` are the shared fallbacks; error
  UI never shows raw error text. Design tokens and styled states come from 0.02.
- **Configuration**: the web app reads only `web/.env.local` and only `NEXT_PUBLIC_*` values;
  the root `.env` (database, OAuth secrets, AI keys) is read by the Go API alone.
- Stable test selectors: `data-testid` on interactive elements used by E2E flows (5.04).

## 5. Mobile

Flutter (one codebase for Android and iOS), built later in `mobile/`. It uses the same API
and accounts; see [mobile/README.md](../mobile/README.md).

## 6. Security model (summary)

| Concern | Decision |
| --- | --- |
| Transport | HTTPS only in production (HSTS header from the API). |
| CORS | Exact web origins from `CORS_ALLOWED_ORIGINS`; `*` is rejected at startup; production requires `https://`. Native apps send no `Origin` — CORS is not authentication. |
| Authorization | In the API: roles/permissions middleware on every Owner route (1.02); learner endpoints filter by the authenticated user (no IDOR). |
| Input | Validated server-side on every endpoint; SQL always parameterised. |
| Errors | Stable codes, no stack traces/SQL/hosts in responses; details go to logs with the request id. |
| Secrets | Environment only; validated at startup; never logged; never in the web bundle. |
| Rate limits | Per-IP default on `/api/v1`; stricter per-route limits on sign-in, codes, AI. |
| Headers | API: nosniff, DENY framing, no-referrer, `default-src 'none'` CSP. Web: nosniff, DENY, frame-ancestors/object-src/base-uri CSP. |

## 7. Decisions

| # | Decision | Why |
| --- | --- | --- |
| 1 | Modular monolith (one Go binary) | Enough for MVP scale; module boundaries keep a later split possible. |
| 2 | API on port **8090**, web on **3000** | 8080 is used by GitPulse on the developer machine. |
| 3 | Separate env files: root `.env` (API) and `web/.env.local` (public only) | The web app cannot accidentally bundle a backend secret. |
| 4 | `/health` = liveness, `/ready` = readiness (DB ping, 503 when down) | Orchestrators restart on liveness, route traffic on readiness. The API starts even if the DB is down and recovers by itself. |
| 5 | pgx + hand-written SQL, no ORM | Explicit, parameterised queries; PostgreSQL features (partial indexes, `ON CONFLICT`) without an abstraction layer. |
| 6 | No custom `.env` library: a 40-line loader in `internal/config` | Avoids a dependency; process environment always wins. |
| 7 | In-memory rate limiter behind an interface | No Redis in the stack yet; swap to a shared store before horizontal scaling. |
| 8 | Prettier from day one | ~40 sessions edit the web app; one formatter avoids style churn in diffs. |
| 9 | Migration files prefixed with a timestamp | Parallel sessions never collide on migration numbers (CLAUDE.md). Tool chosen by 0.01. |
| 10 | Flutter for mobile; OpenAPI is the shared contract | TypeScript types cannot be shared with Dart; a generated client keeps mobile in sync. |

## 8. Open questions (need a decision before the owning session starts)

- **Session model (1.01)**: recommended in [AUTHENTICATION.md](AUTHENTICATION.md) — short JWT
  access token + rotating opaque refresh token for every client. 1.01 confirms.
- **Owner MFA (1.02)**: TOTP + recovery codes recommended.
- **Production hosting, S3 provider, email provider (5.05 / 1.01)**: not decided.
- **Multi-instance rate limiting (5.05)**: Redis vs PostgreSQL-backed limiter.
