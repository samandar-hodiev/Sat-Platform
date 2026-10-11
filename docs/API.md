# API

One REST API serves the learner web app, the Owner dashboard and the Android/iOS apps. These
conventions apply to every module; each session adds its endpoints to the table at the end
and to `api/openapi/openapi.yaml` in the same change.

Base URL (local): `http://localhost:8090` · Machine-readable contract: `GET /api/v1/openapi.yaml`

## Versioning and compatibility

- Business endpoints live under **`/api/v1`**. Only the probes `/health` and `/ready` sit outside.
- `/api/v1` is **additive-only** once a mobile app ships: new endpoints, new optional request
  fields and new response fields are fine. Removing or renaming a field, changing its type or
  meaning, or changing an error code is breaking.
- Breaking changes go to `/api/v2` (new handlers/DTOs over the same services); `v1` stays until
  the app versions that use it are retired.
- Clients ignore unknown fields and unknown enum values.

## Requests

- JSON (`Content-Type: application/json`), UTF-8, `snake_case` field names. Uploads use
  `multipart/form-data` on dedicated routes with their own size limits.
- Default body limit 1 MiB (`HTTP_MAX_BODY_BYTES`) → `413 PAYLOAD_TOO_LARGE`.
- All input is validated on the server, whatever the client validated.

| Header | Purpose |
| --- | --- |
| `Authorization: Bearer <access_token>` | Authentication ([AUTHENTICATION.md](AUTHENTICATION.md)) |
| `X-Client-Platform: web \| android \| ios` | Diagnostics and session metadata only; never changes behaviour |
| `X-Request-ID` | Optional trace id (8–64 chars `[A-Za-z0-9._-]`); generated when absent; echoed in the response header and error body |

## Responses

```json
{ "success": true, "data": { "id": "…" } }
```

```json
{ "success": true, "data": [ … ], "meta": { "page": 1, "page_size": 20, "total": 134 } }
```

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Some fields are invalid",
    "details": { "fields": { "email": "must be a valid email address" } },
    "request_id": "3d6be0080317844f908d687ceac3e82b"
  }
}
```

- `204 No Content` has no body.
- Clients branch on `error.code` (stable). `message` is human-readable English and may change.
- Responses never include stack traces, SQL, hostnames or provider errors.

### Status codes and error codes

| HTTP | `error.code` | When |
| --- | --- | --- |
| 200 / 201 / 204 | — | OK / created / done, no body |
| 400 | `BAD_REQUEST` | Malformed JSON or parameters |
| 401 | `UNAUTHORIZED` | Missing, invalid or expired access token |
| 403 | `FORBIDDEN` | Authenticated but not allowed (role, permission, plan feature) |
| 404 | `NOT_FOUND` | Unknown route, or a resource that does not exist *or belongs to someone else* |
| 405 | `METHOD_NOT_ALLOWED` | Wrong method on an existing route |
| 409 | `CONFLICT` | Uniqueness or state conflict (`details.reason` names it) |
| 413 | `PAYLOAD_TOO_LARGE` | Body or upload too large |
| 422 | `VALIDATION_ERROR` | Field rules failed; `details.fields` |
| 429 | `RATE_LIMITED` | Too many requests; `Retry-After` header and `details.retry_after_seconds` |
| 500 | `INTERNAL_ERROR` | Unexpected server error (logged with the request id) |
| 503 | `SERVICE_UNAVAILABLE` | A dependency is down |

New codes (e.g. `PLAN_REQUIRED` for Premium gating in 3.13) are added to `apperr`, to
`openapi.yaml` and to `web/src/types/api.ts` together; a test checks the first two match.

## Pagination, filtering, sorting

- `?page=1&page_size=20` (`page_size` max 100) → `meta.page`, `meta.page_size`, `meta.total`
  (`httpx.Pagination`).
- Filters are query parameters named after the field: `?domain=algebra&difficulty=hard`.
  Several values repeat the parameter: `?status=draft&status=in_review`.
- Sorting: `?sort=<key>` from the endpoint's documented allow-list (validated with
  `binding:"omitempty,oneof=…"`), `-` prefix for descending: `?sort=-created_at`. Unknown keys
  are `422`, never silently ignored. Default order is documented per endpoint.

## Dates, numbers, identifiers

- Timestamps: RFC 3339 in **UTC** with `Z` — `2026-10-11T10:45:22Z`. Clients show local time.
- Dates without time (exam date, plan day): `YYYY-MM-DD`.
- Durations: integers with the unit in the name — `time_limit_seconds`, `latency_ms`.
- The learner's IANA timezone is stored on the profile; day-based logic (daily plan, streaks)
  runs on the server in that timezone.
- IDs: UUID strings, opaque to clients.
- Money: integer minor units + ISO 4217 currency (`amount_minor`, `currency`).
- Scores are practice estimates; the API never presents them as official SAT scores.

## Security rules for endpoints

- Learner endpoints act only on the authenticated user's data (filter by user id from the
  token, never from the request body).
- Every `/api/v1/owner/*` route requires the owner role/permission in the API (1.02).
- Exam and practice endpoints never send correct answers before submission (3.06, 3.08).
- Rate-limit sensitive endpoints with their own scope (sign-in, code send/verify, reset, AI).

## Endpoints

| Method | Path | Auth | Session | Description |
| --- | --- | --- | --- | --- |
| GET | `/health` | – | 0.00 | Liveness: the process is running |
| GET | `/ready` | – | 0.00 | Readiness: PostgreSQL answers (503 otherwise) |
| GET | `/api/v1/openapi.yaml` | – | 0.00 | OpenAPI 3.1 contract |
