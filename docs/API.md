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
| … | `/api/v1/me/profile*`, `/api/v1/me/avatar`, `/api/v1/media/avatars/*` | Bearer / – | 2.01 | Learner profile (below) |
| … | `/api/v1/me/onboarding*` | Bearer | 2.02 | Learner onboarding answers: goal, schedule, background (below) |

### Auth — learner (1.01)

Status: live (`api/internal/auth`, PostgreSQL store, mounted in `server.NewRouter`). Also in
`openapi.yaml` (tag `auth`). Token model: [AUTHENTICATION.md](AUTHENTICATION.md).

| Method | Path | Auth | Per-IP limit | Description |
| --- | --- | --- | --- | --- |
| POST | `/api/v1/auth/sign-up/start` | – | 5/min | Email a 6-digit code (same answer for new and existing emails) |
| POST | `/api/v1/auth/sign-up/verify` | – | 10/min | Verify the code → session. New email: account created (`is_new_user: true`); existing learner: signed in |
| POST | `/api/v1/auth/sign-in` | – | 10/min (+10 per email / 15 min) | Email + password → session |
| POST | `/api/v1/auth/google` | – | 10/min | Google ID token → session (`503` when `GOOGLE_CLIENT_IDS` is empty) |
| POST | `/api/v1/auth/refresh` | – | 30/min | Rotate the refresh token → new session pair |
| POST | `/api/v1/auth/logout` | – | – | Revoke the session of the refresh token (`204`, idempotent) |
| POST | `/api/v1/auth/password/forgot` | – | 5/min (+3 per email / hour, silent) | Email a reset link (`204` always) |
| POST | `/api/v1/auth/password/reset` | – | 10/min | Set a new password from the link (`204`); signs out every session |
| GET | `/api/v1/auth/me` | Bearer | – | The signed-in user |

**Requests**

```jsonc
// sign-up/start, password/forgot
{ "email": "learner@example.com" }
// sign-up/verify
{ "email": "learner@example.com", "code": "123456" }
// sign-in
{ "email": "learner@example.com", "password": "…", "remember": true }
// google
{ "id_token": "<Google ID token from the platform SDK>" }
// refresh, logout
{ "refresh_token": "…" }
// password/reset
{ "token": "<from the emailed link>", "password": "8–128 characters" }
```

**Responses**

```jsonc
// sign-up/start → 200
{ "expires_at": "2026-10-12T09:10:00Z", "resend_available_at": "2026-10-12T09:01:00Z" }

// sign-up/verify, sign-in, google, refresh → 200 (the "session" object)
{
  "access_token": "…", "access_token_expires_at": "2026-10-12T09:15:00Z",
  "refresh_token": "…", "refresh_token_expires_at": "2026-11-11T09:00:00Z",
  "token_type": "Bearer", "remember": true, "is_new_user": false,
  "user": { "id": "uuid", "email": "learner@example.com", "role": "learner",
            "email_verified": true, "has_password": false, "created_at": "…" }
}

// me → 200 (same user object plus sign-in methods)
{ "user": { /* … */, "has_google": true } }
```

**Errors** (messages are written for learners and shown as is)

| Case | Status / code | Notes |
| --- | --- | --- |
| Wrong email or password, unknown email, account without password | `401 UNAUTHORIZED` | One message for all: "Email or password is incorrect." |
| Suspended/deleted account, owner account on learner routes | `403 FORBIDDEN` | Owners use `/owner/login` (1.02) |
| Wrong / expired / used code, too many attempts | `422 VALIDATION_ERROR` | `details.fields.code` ("The code is incorrect. 4 attempts left.") |
| Code requested too soon / daily cap (10 per email per 24 h) | `429 RATE_LIMITED` | `details.retry_after_seconds` for the cooldown |
| Refresh token unknown, rotated, revoked or expired | `401 UNAUTHORIZED` | Reuse of a rotated token after 30 s revokes the whole session |
| Access token expired | `401 UNAUTHORIZED` | `details.reason: "token_expired"` → refresh and retry |
| Invalid or used reset link | `400 BAD_REQUEST` | "This reset link is invalid or has expired." |
| Email or Google not configured | `503 SERVICE_UNAVAILABLE` | `MAIL_DRIVER=disabled`, empty `GOOGLE_CLIENT_IDS` |

Other modules protect learner routes with `authModule.Authenticate()` and read the user with
`auth.CurrentUser(c)` (`UserID`, `Role`, `SessionID`). Go-only helpers for other modules (no HTTP
route): `Service.SetInitialPassword` / `SetInitialPasswordTx(ctx, auth.PgxTx(tx), …)` (first
password; `409` if one exists), `Service.Account(ctx, userID)` (user + `HasGoogle`,
`HasPassword()`) and the `auth.ProfilePrefiller` hook (new Google learner's name).

**Web only — `/api/session/*` (Next.js route handlers, same origin):** `sign-in`,
`sign-up/verify`, `google`, `refresh`, `logout`. Same bodies as above, but the refresh token
never appears in the response: it is set as an HttpOnly cookie. Every request needs the header
`X-Session-Request: 1`. Mobile apps call `/api/v1/auth/*` directly.

### Profile — learner (2.01)

Status: live (`api/internal/profile`, mounted in `server.NewRouter` after auth). Also in
`openapi.yaml` (tag `profile`). Learner role only (`403` for other roles).

| Method | Path | Auth | Limit | Description |
| --- | --- | --- | --- | --- |
| GET | `/api/v1/me/profile` | Bearer | – | The learner's profile (empty and incomplete right after sign-up) |
| PUT | `/api/v1/me/profile/setup` | Bearer | – | Finish Profile Setup once; sets the first password when given |
| POST | `/api/v1/me/avatar` | Bearer | 20 / learner / hour | Upload a picture (multipart field `file`) |
| DELETE | `/api/v1/me/avatar` | Bearer | – | Back to the default avatar |
| GET | `/api/v1/media/avatars/{user}/{file}` | – | – | The stored image (public; 128 random bits in the URL; `Cache-Control: immutable`) |

```jsonc
// GET /me/profile, PUT /me/profile/setup, POST|DELETE /me/avatar → 200
{
  "email": "learner@example.com",          // verified email, read-only here
  "first_name": "Ali", "last_name": "Valiyev",
  "phone": "+998901234567",                // E.164 or null
  "date_of_birth": "2008-05-01",           // or null
  "avatar_url": "/api/v1/media/avatars/<user-id>/<32 hex>.jpg", // path on the API origin; null = default avatar
  "timezone": "Asia/Tashkent",
  "has_password": true,
  "password_required": false,              // true: Profile Setup must set a password
  "profile_completed": true,               // false → clients send the learner to Profile Setup
  "profile_completed_at": "2026-10-12T09:00:00Z"
}

// PUT /me/profile/setup
{ "first_name": "Ali", "last_name": "Valiyev", "phone": "+998 90 123 45 67",
  "date_of_birth": "2008-05-01", "timezone": "Asia/Tashkent", "password": "…" }
```

Rules (all field errors come back at once in `details.fields`):

- Names: 1–100 characters; letters of any script with spaces, `-`, `'`, `’`, `.`; whitespace collapsed.
- Phone: optional; typed with spaces, dashes, brackets or a `00` prefix; stored as E.164.
- Date of birth: required, `YYYY-MM-DD`, from 1900, learner at least **13** today (UTC).
- Timezone: optional IANA name (the web sends the browser's); empty keeps the current one.
- Password: required when `password_required` (email account without one), optional for Google
  accounts, rejected when `has_password`. Auth password policy (8–128). Saved in the same
  transaction as the profile.
- Avatar: JPG, PNG or WebP detected from the bytes (file name and Content-Type ignored), ≤ 5 MB,
  ≤ 40 MP. Stored as JPEG: centre square, at most 512×512, EXIF orientation applied, all metadata
  dropped, transparency on white. A new upload replaces (and deletes) the previous file.

| Case | Status / code | Notes |
| --- | --- | --- |
| Profile Setup already finished | `409 CONFLICT` | `details.reason: "profile_already_completed"`; later edits: `PATCH /me/profile` (My Profile, 2.10) |
| Invalid fields, bad image | `422 VALIDATION_ERROR` | `details.fields.<field>`; image errors on `file` |
| Upload body over 5 MB + framing | `413 PAYLOAD_TOO_LARGE` | |
| Not multipart / unreadable upload | `400 BAD_REQUEST` | |
| Session ended / account gone | `401 UNAUTHORIZED` | |

Go: `profile.GooglePrefiller` implements `auth.ProfilePrefiller` (fills empty names of a new
Google learner); `profile.AuthAccounts` adapts `auth.Service`. The upload route has its own body
limit through `middleware.BodyLimitWithRoutes`.

### My Profile — learner (2.10)

Status: live. Email, password, sign-in methods: `api/internal/auth` (`account.go`,
`handler_account.go`); profile edit and account deletion: `api/internal/profile` (`account.go`,
`handler_account.go`). Also in `openapi.yaml` (tag `account`). Learner role only (`403` for other
roles). PRD §20 Account.

| Method | Path | Auth | Limit (per learner) | Description |
| --- | --- | --- | --- | --- |
| PATCH | `/api/v1/me/profile` | Bearer | – | Edit the profile after Profile Setup; returns the profile (shape as in Profile) |
| GET | `/api/v1/me/sign-in-methods` | Bearer | – | Email, password, Google, and what sensitive changes ask for |
| POST | `/api/v1/me/reauth/code` | Bearer | 5 / 10 min | Email a confirmation code (accounts **without** a password; `409` otherwise) |
| POST | `/api/v1/me/email/change` | Bearer | 10 / 10 min | Confirm the learner, then send a code to `new_email` |
| POST | `/api/v1/me/email/change/confirm` | Bearer | 10 / 10 min | `{ code }` → the email changes |
| PUT | `/api/v1/me/password` | Bearer | 10 / 10 min | Change the password, or set the first one → `204` |
| POST | `/api/v1/me/sign-in-methods/google` | Bearer | 10 / 10 min | Link a Google account (`id_token` + confirmation) |
| DELETE | `/api/v1/me/sign-in-methods/google` | Bearer | – | Unlink it |
| DELETE | `/api/v1/me/account` | Bearer | 10 / 10 min | Delete the account now → `204` |

```jsonc
// PATCH /me/profile — absent or null fields stay; phone "" removes the phone
{ "first_name": "Aziza", "phone": "", "timezone": "Europe/London" }

// GET /me/sign-in-methods, POST /me/email/change/confirm, POST|DELETE /me/sign-in-methods/google → 200
{
  "email": "learner@example.com", "email_verified": true,
  "has_password": true,
  "email_code": true,                       // a code sent to the email always signs in
  "reauth_method": "password",              // "password" | "email_code"
  "google": { "email": "learner@gmail.com", "linked_at": "2026-10-17T09:00:00Z" } // or null
}

// Confirmation ("reauth") fields, part of every sensitive request below: send the one that applies
{ "current_password": "…" }   // reauth_method "password"
{ "code": "123456" }          // reauth_method "email_code" (from POST /me/reauth/code)

// POST /me/email/change
{ "new_email": "new@example.com", "current_password": "…" }
// → 200 { "new_email": "new@example.com", "expires_at": "…", "resend_available_at": "…" }

// PUT /me/password
{ "new_password": "…", "current_password": "…" }

// POST /me/sign-in-methods/google
{ "id_token": "<Google ID token>", "code": "123456" }

// DELETE /me/account
{ "confirmation": "DELETE", "current_password": "…" }
```

Rules (decided for 2.10):

- **Confirmation:** accounts with a password confirm with it (5 tries per 15 min per learner, then
  `429`); accounts without one confirm with a 6-digit code mailed to the current email (10 min,
  5 wrong guesses, 60 s between codes, 10 per day). A code is spent when accepted. Codes are bound
  to their purpose: a confirmation code is never a sign-up or email-change code.
- **Email change:** the learner is confirmed first, then the new address is checked (`422`
  `new_email` when another live account has it — also re-checked on confirm). The code goes to the
  new address; on confirm the email changes, `email_verified` is true, other sessions are signed
  out, and the old address gets a notice.
- **Password:** policy 8–128 (auth). Must differ from the current one. Other sessions are signed out;
  the current one stays. Accounts without a password set their first one here, confirmed by code.
- **Google:** linking needs confirmation (a linked Google account is a way in); `409` when one is
  linked already or the Google account belongs to another learner. Unlinking never locks the learner
  out (the email code always works). Note: Google sign-in (1.01) links a Google account whose email
  equals the account email automatically, so such an account is linked again on its next Google
  sign-in.
- **Account deletion:** `confirmation` must be exactly `DELETE`, plus the confirmation fields.
  Immediate and final (soft delete, no grace period): every session ends, the email is freed, the
  password, identities and pending codes/links are removed, and the profile row (name, phone, date
  of birth) and avatar file are deleted — one transaction for the database part. Not covered yet:
  cancelling a paid subscription (3.13/5.01) and an audit log entry (1.02 helper) — see PROGRESS.
- **Profile edit:** only after Profile Setup (`409`, `details.reason: "profile_setup_required"`);
  same field rules as Profile Setup (names, phone, date of birth 13+, IANA timezone, which may not
  be empty here). All field errors at once in `details.fields`.

| Case | Status / code | Notes |
| --- | --- | --- |
| Wrong/missing password or code, invalid field | `422 VALIDATION_ERROR` | `details.fields.current_password`, `code`, `new_email`, `new_password`, `confirmation`, profile fields |
| Code requested while the account has a password | `409 CONFLICT` | |
| Google already linked / owned by another learner | `409 CONFLICT` | |
| Unlink with nothing linked | `404 NOT_FOUND` | |
| Too many password tries, codes or requests | `429 RATE_LIMITED` | `details.retry_after_seconds` for the code cooldown |
| Email or Google not configured | `503 SERVICE_UNAVAILABLE` | |
| Session ended, account deleted | `401 UNAUTHORIZED` | Access tokens of a deleted account stop working on every endpoint that loads the account |

### Onboarding — learner (2.02)

Status: live (`api/internal/onboarding`, mounted in `server.NewRouter` after profile). Also in
`openapi.yaml` (tag `onboarding`). Learner role only (`403` for other roles). PRD §8.

Three steps, each saved on its own and editable any time, also after onboarding is finished
(Learning Preferences, 2.11, uses the same `PUT`s). `POST /complete` ends onboarding; the web then
continues to Placement (3.11; until it exists, `/app`). The web blocks `/app/*` until `completed`.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/api/v1/me/onboarding` | Bearer | Current answers and state (works before Profile Setup; all steps `null`) |
| PUT | `/api/v1/me/onboarding/goal` | Bearer | Step 1: target score, exam date, reason |
| PUT | `/api/v1/me/onboarding/schedule` | Bearer | Step 2: daily minutes, study days |
| PUT | `/api/v1/me/onboarding/background` | Bearer | Step 3: confidence, earlier SAT result |
| POST | `/api/v1/me/onboarding/complete` | Bearer | Finish (every step saved). Idempotent: keeps the first `completed_at` |

```jsonc
// every endpoint → 200. A step is null until saved: "not set" is never a default value.
{
  "completed": false,
  "completed_at": null,                    // RFC 3339 once finished
  "next_step": "schedule",                 // first unsaved step: goal | schedule | background; null when all saved
  "goal": {
    "target_score": 1450,                  // 400–1600, steps of 10
    "exam_date": "2027-03-13",             // null = the learner has not decided yet
    "days_until_exam": 151,                // from the learner's today (profile time zone); null without a date; negative once passed
    "reason": "scholarship",               // college_admission | scholarship | improve_score | practice | other | null
    "saved_at": "2026-10-12T09:00:00Z"
  },
  "schedule": null,                        // { "daily_minutes": 45, "study_days": [1, 3, 5], "saved_at": … }
  "background": null                       // { "math_confidence": 3, "reading_writing_confidence": 4,
                                           //   "previous_score": { "total": 1200, "math": 600, "reading_writing": 600, "taken_month": "2026-05" } | null,
                                           //   "saved_at": … }
}

// PUT /me/onboarding/goal
{ "target_score": 1450, "exam_date": "2027-03-13", "reason": null }
// PUT /me/onboarding/schedule
{ "daily_minutes": 45, "study_days": [5, 1, 3] }
// PUT /me/onboarding/background
{ "math_confidence": 3, "reading_writing_confidence": 4,
  "previous_score": { "total": 1200, "math": 600, "reading_writing": 600, "taken_month": "2026-05" } }
```

Rules (all field errors come back at once in `details.fields`; earlier-score fields as
`previous_score.<field>`):

- `target_score`: required, 400–1600, multiple of 10.
- `exam_date`: `YYYY-MM-DD`, from the learner's today (profile time zone) to 2 years ahead; `null`
  or omitted = undecided.
- `reason`: optional, one of the values above.
- `daily_minutes`: required, one of 15, 30, 45, 60, 90, 120 (120 = "2+ hours").
- `study_days`: required, ISO weekdays 1 (Mon) … 7 (Sun), at least one; repeats dropped, sorted.
- `math_confidence`, `reading_writing_confidence`: required, 1–5.
- `previous_score`: optional (`null` = none / skipped; saving without it clears it). `total`
  required 400–1600 ×10; `math` and `reading_writing` optional but together, 200–800 ×10, adding
  up to `total`; `taken_month` optional `YYYY-MM`, March 2016 (1600 scale) to this month.
- Each `PUT` replaces the whole step (omitted optional fields become `null`).

| Case | Status / code | Notes |
| --- | --- | --- |
| Profile Setup not finished (writes) | `409 CONFLICT` | `details.reason: "profile_setup_required"` |
| `complete` before every step is saved | `409 CONFLICT` | `details.reason: "onboarding_incomplete"`, `details.missing_steps: ["schedule", …]` |
| Invalid fields | `422 VALIDATION_ERROR` | `details.fields.<field>` |
| Malformed JSON, wrong types (e.g. `"1400"`, `1400.5`) | `400 BAD_REQUEST` | |
| Session ended / account gone | `401 UNAUTHORIZED` | |

Go: other modules read the answers with `onboarding.Module.Service.Get(ctx, userID)` (study plan
2.05, dashboard 2.04); `onboarding.ProfileService` adapts `profile.Service` (Profile Setup done,
time zone). Web: `useOnboarding()` from `@/features/onboarding` (one fetch per session, shared).
