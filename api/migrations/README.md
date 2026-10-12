# Migrations

PostgreSQL schema changes live here and nowhere else. Tool: golang-migrate, embedded in
`cmd/migrate` (this folder is compiled in through `embed.go`). Full rules: [docs/DB.md](../../docs/DB.md).

```bash
make migrate-create name=create_learner_goals   # new empty up/down pair with a timestamp
make migrate-up                                  # apply pending migrations (DATABASE_URL)
make migrate-status
make migrate-down                                # revert the last one (n=3 for more)
```

- One pair per change: `YYYYMMDDHHMMSS_snake_case_name.up.sql` / `.down.sql`. The timestamp
  prefix keeps parallel sessions from colliding on numbers.
- Wrap statements in `BEGIN; … COMMIT;` (except `CREATE INDEX CONCURRENTLY`, alone in its file).
- Never edit a migration that has been applied anywhere; add a new one.
- `down` reverts `up` exactly. Re-running `up` on an up-to-date database changes nothing.
- `go test ./internal/platform/migrate` checks names and pairs; with `TEST_DATABASE_URL` it also
  runs up → down → up and the schema constraint tests.
