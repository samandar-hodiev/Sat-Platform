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
│   ├── cmd/migrate/main.go       migration CLI: up | down [n] | status | force | create (0.01)
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
│   │   │   ├── migrate/          golang-migrate wrapper + file-name validation (0.01)
│   │   │   ├── logger/           slog (text in dev, JSON in prod)
│   │   │   ├── ratelimit/        Limiter interface + in-memory implementation
│   │   │   └── storage/          ObjectStorage interface + local implementation
│   │   └── <module>/             one folder per domain, added by its session (auth, users, …)
│   ├── openapi/                  openapi.yaml (embedded and served) + contract tests
│   └── migrations/               SQL migrations, embedded into cmd/migrate (embed.go)
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
  | `(focus)` | `/exam/*` — full-screen mock exam runner without the sidebar (layout: 2.03) | 3.08 |
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

### 4.1 Design system (0.02)

- **Tokens** live in `web/src/app/globals.css` and are the only allowed colors, radii, shadows,
  font sizes, z-indexes and durations. Tailwind's default palette is reset, so `bg-blue-500` or
  `text-gray-600` produce nothing; arbitrary values (`bg-[#…]`, `p-[13px]`) are not used.
  Spacing uses Tailwind's 4px scale (`p-4`, `gap-3`).
- **Color utilities** (theme-aware, no `dark:` prefixes needed): `bg-canvas` (page), `bg-surface`,
  `bg-surface-muted`, `bg-surface-hover`; `text-fg`, `text-fg-muted`, `text-fg-subtle`;
  `border-border`, `border-border-strong`; `bg-primary` + `text-primary-fg`, `bg-primary-soft`,
  `text-primary-text`; `bg-secondary` + `text-secondary-fg`; `success|warning|error|info` with
  `-soft` (background) and `-text` (text on soft or surface); `bg-error` + `text-error-fg`;
  `chart-1…5`, `chart-grid`, `chart-axis` for charts. `tokens.test.ts` enforces WCAG AA contrast
  for these pairs in every theme and accent.
- **Other tokens**: `text-xs…4xl`, `rounded-sm|md|lg|xl|full`, `shadow-xs|sm|md|lg`,
  `h-(--control-sm|md|lg)` for control heights (density-aware), `z-(--z-dropdown|modal|toast|tooltip|header|drawer|sticky)`,
  `duration-(--duration-fast|base|slow)` (0 with reduced motion), `ease-standard`,
  `animate-fade-in|pop-in|slide-in|spin|pulse`, `focus-ring` utility.
- **Theme**: attributes on `<html>` — `data-theme` (light|dark, resolved), `data-accent`
  (neutral|blue|indigo|violet|teal|rose), `data-density` (comfortable|compact), `data-motion`
  (system|reduced|full). An inline script (`src/lib/theme/script.ts`) applies them before paint.
  Default: neutral Light (PRD §4.2).
- **Preferences** are stored in a first-party cookie per scope: `sat_learner_appearance` and
  `sat_owner_appearance` (`/owner/*`), so Owner and Learner choices are independent (PRD §21.3).
  Not in `localStorage` (ESLint rule). `useTheme()` from `@/components/theme/ThemeProvider` reads
  and changes them; `setPreferences()` accepts values loaded from the account — DB sync and the
  Appearance page are 2.11, wallpaper is 2.11, exam focus mode (no decorations) is 3.08.
- **UI kit**: `@/components/ui` — Button/`buttonClasses` (for links), IconButton, FormField +
  Input/Textarea/Select (label, hint, error and aria wiring), Checkbox, Radio, ChoiceGroup,
  Switch, Card*, Badge, Avatar, Tabs*, Accordion*, Modal (native `<dialog>`), DropdownMenu*,
  Tooltip, ToastProvider/`useToast`, Spinner, Skeleton/SkeletonText, LoadingState, EmptyState,
  ErrorState, and a small icon set (`icons.tsx`). `ThemeSwitcher` (Light/Dark/System) is in
  `@/components/theme`. Feature components compose these; they do not restyle them with colors.
- **Catalogue**: `/dev/ui-kit` in development (`npm run dev`). Production 404s every `/dev/*`
  path (rewrite in `next.config.ts`).
- Component tests run in jsdom (`// @vitest-environment jsdom`, Testing Library);
  `src/test/setup.ts` adds cleanup, `matchMedia` and `<dialog>` shims.

### 4.2 Learner shell (2.03)

- `app/(learner)/app/layout.tsx` wraps every `/app/*` page in `LearnerShell`
  (`@/features/learner-shell`): collapsible sidebar (lg and up), sticky header, `<main id="main">`,
  skip link. Below lg the same navigation opens in a native `<dialog>` drawer from the header.
- **Navigation** is one config: `features/learner-shell/nav.ts` (`LEARNER_NAV`, PRD §10.2 groups
  Learn / Track / Account). A session that builds a page creates `app/(learner)/app/<path>/page.tsx`
  at the href listed there; it does not edit the sidebar. Paths: `/app` Dashboard, `/app/plan`,
  `/app/learn`, `/app/exams`, `/app/library`, `/app/assistant`, `/app/progress`, `/app/saved`,
  `/app/mistakes`, `/app/profile`, `/app/settings`, `/app/subscription`. Nested pages
  (`/app/exams/<id>`) highlight their section automatically.
- **Unbuilt pages**: `app/(learner)/app/[...missing]` shows the in-shell "Page not found"; real
  routes always take precedence. `(learner)/app/error.tsx` and `loading.tsx` keep the shell.
- **Header**: section breadcrumb (pages render their own `<h1>`), an actions slot for
  notifications (3.14) and streak/progress (2.04, 2.07), then the profile menu (My Profile,
  Settings, Subscription, Sign out).
- **Account data**: `useShellAccount()` in `features/learner-shell/account.ts` returns
  `loading | unavailable | ready{user, signOut}`. Until the 1.01 web session exists it returns
  `unavailable`: no name is shown and Sign out is disabled. 1.01 replaces only that hook.
- **Sidebar state**: cookie `sat_learner_sidebar` (`expanded|collapsed`), read on the server so
  the first paint has the right width. 2.11 may sync it with account preferences.
- **Focus layout**: `app/(focus)/exam/layout.tsx` — no sidebar, header or decorations, for the
  exam runner (3.08 adds `/exam/[attemptId]`; until then `/exam/*` is not found in that layout).
- Pages under `/app` and `/exam` are `noindex`.

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
| 9 | Migration files prefixed with a timestamp; golang-migrate embedded in `cmd/migrate`; the server never migrates on start-up | Parallel sessions never collide on migration numbers (CLAUDE.md). Plain SQL, version table and advisory lock; a separate migrate step keeps several API instances from racing on start-up. |
| 10 | Flutter for mobile; OpenAPI is the shared contract | TypeScript types cannot be shared with Dart; a generated client keeps mobile in sync. |

## 8. Open questions (need a decision before the owning session starts)

- ~~**Session model (1.01)**~~: decided — short JWT access token + rotating opaque refresh token
  for every client ([AUTHENTICATION.md](AUTHENTICATION.md)).
- **Owner MFA (1.02)**: TOTP + recovery codes recommended.
- **Production hosting, S3 provider, email provider (5.05 / 1.01)**: not decided.
- **Multi-instance rate limiting (5.05)**: Redis vs PostgreSQL-backed limiter.
