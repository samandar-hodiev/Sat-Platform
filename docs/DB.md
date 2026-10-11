# Database

PostgreSQL is the single database for the web app, the Owner dashboard and the Android/iOS
apps. Session **0.01** adds the ERD, the migration tool and the first tables to this file;
this part (by 0.00) fixes the rules every migration follows.

Local development uses PostgreSQL 18 (Postgres.app); production version is decided by 5.05
(PostgreSQL 15+ is required).

## Access from Go

- One `pgxpool.Pool` (`internal/platform/database`) is created at startup and passed to
  modules. Pool size: `DATABASE_MAX_CONNS`.
- SQL is hand-written in each module's repository with `$1, $2` parameters. Identifiers are
  never built from user input. No ORM.
- `GET /ready` pings the database; the API starts even when PostgreSQL is down and becomes
  ready once it answers.

## Migrations

- Location: `api/migrations/`. File names: `YYYYMMDDHHMMSS_snake_case_name.up.sql` and
  `.down.sql`. Timestamps (not 0001, 0002…) so sessions working in parallel never collide.
- Tool: chosen and wired by 0.01 (golang-migrate recommended: plain SQL files, a version
  table, `up`/`down`/`force`, and it can be embedded in the API binary).
- **Re-running is safe**: the tool records applied versions, so `up` on an up-to-date database
  does nothing and never touches data.
- An applied migration is never edited — fix forward with a new one.
- `down` must exactly revert `up`; an integration test runs up → down → up on a disposable
  database (`TEST_DATABASE_URL`, never the development database).
- Keep a migration atomic: wrap dependent statements in `BEGIN; … COMMIT;`.
  `CREATE INDEX CONCURRENTLY` cannot run in a transaction and goes in its own migration.
- **Expand → migrate → contract**: add the new column/table first, backfill, switch the code,
  drop the old column in a later release. Old mobile app versions keep working meanwhile.
- Reference rows every environment needs (roles, SAT domains, plans) are inserted with
  `ON CONFLICT DO NOTHING`. Demo/sample data never goes into migrations; it lives in a
  development-only seed command.

## Naming

- Tables: plural `snake_case` (`exam_attempts`, `question_versions`); join tables name both
  sides (`plan_features`).
- Columns: `snake_case`; foreign keys `<entity>_id`; booleans `is_…`/`has_…`; timestamps `…_at`.
- Primary keys: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`. Lookup tables may use a
  natural text key (`code`).
- Indexes `<table>_<columns>_idx`, unique indexes `<table>_<columns>_key`.
- Enum-like values: lowercase `snake_case` text with a `CHECK` constraint (easier to extend than
  PostgreSQL enums).

## Integrity

- Every relationship has a foreign key. `ON DELETE CASCADE` for data owned by a user (attempts,
  bookmarks, mistakes); `RESTRICT` or `SET NULL` where history must survive (a retired question
  version stays referenced by old exam responses).
- Invariants are constraints, not just application checks: `NOT NULL`, `UNIQUE`, `CHECK`, and
  partial unique indexes for "only one active …" (one active study plan, one in-progress exam
  attempt per exam).
- Every mutable table has `created_at` and `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`;
  `updated_at` is maintained by a shared trigger (0.01). `updated_at` also enables incremental
  mobile sync later (`?updated_since=`).
- Soft delete (`deleted_at`) only where history or recovery matters (users pending deletion,
  published content). Unique constraints on soft-deleted tables are partial
  (`WHERE deleted_at IS NULL`).
- Content that learners answer against is versioned (`question_versions`); attempts reference
  the exact version they saw.
- Audit: `audit_logs` is append-only (no UPDATE/DELETE from the application) and records actor,
  action, entity, metadata and time for every important Owner action.

## Transactions

- A business operation that writes more than one row runs in one transaction inside the owning
  module's repository/service (`pool.Begin(ctx)` + deferred `Rollback`, `Commit` at the end).
  Handlers never manage transactions.
- Counters and limits use one conditional `UPDATE … RETURNING`, not read-modify-write.
- Retries must be safe: submissions, webhooks and job results use a unique key with
  `ON CONFLICT` (idempotency). Exam submit is idempotent (3.08).

## Planned entities (PRD §3)

Users, Profiles, User Preferences, Learner Goals, Study Plans, Study Plan Tasks, Curriculum
Domains, Topics, Lessons, Questions, Question Versions, Practice Attempts, Mock Exams, Exam
Modules, Exam Attempts, Exam Responses, Bookmarks, Mistake Records, AI Conversations,
Subscriptions, Payments, Notifications, Audit Logs. Each is created by the session that owns
it (see CLAUDE.md); the ERD is maintained here by 0.01.
