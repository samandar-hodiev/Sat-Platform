# Migrations

PostgreSQL schema changes live here and nowhere else. The tool, the first tables (users,
profiles, user_preferences, audit_logs) and the `make migrate-*` commands are added by
session **0.01-Foundation · Database & Migrations**.

Rules (full version in [docs/DB.md](../../docs/DB.md)):

- One pair per change: `YYYYMMDDHHMMSS_snake_case_name.up.sql` / `.down.sql`. The timestamp
  prefix keeps parallel sessions from colliding on numbers.
- Never edit a migration that has been applied anywhere; add a new one.
- `down` reverts `up` exactly. Re-running `up` on an up-to-date database changes nothing.
