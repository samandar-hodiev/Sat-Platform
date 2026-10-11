# SAT Platform mobile (planned)

Android and iOS apps will be **one Flutter codebase** in this folder. Nothing is built yet;
this file records what the apps must follow so they fit the backend from day one.

## Same backend, same data

- The apps use the **same Go API, the same PostgreSQL database and the same accounts** as the
  web app. No mobile backend, no mobile database, no mobile-only business rules.
- Base URL `https://<api-host>/api/v1`; every convention in [docs/API.md](../docs/API.md)
  applies (envelope, `error.code`, pagination, UTC RFC 3339 dates, additive-only `v1`).
- The API decides access and outcomes (Premium features, exam timing and scoring, study
  plan); the app displays what it returns. Progress started on the web continues on the phone.
- Generate the Dart client from the contract instead of writing models by hand:
  `openapi-generator generate -i api/openapi/openapi.yaml -g dart-dio -o mobile/lib/api`
  (the spec is also served at `GET /api/v1/openapi.yaml`).

## Authentication

Full design: [docs/AUTHENTICATION.md](../docs/AUTHENTICATION.md).

- Call `/api/v1/auth/*` directly — no cookies, no WebView sessions, no `localStorage`.
- Refresh token in `flutter_secure_storage` (Keychain / Android Keystore); access token in memory.
- Every request: `Authorization: Bearer …` and `X-Client-Platform: android|ios`.
- On `401`: refresh once (single-flight), retry, sign out locally if refresh fails.
- Google: `google_sign_in` → ID token → `POST /api/v1/auth/google`. Create Android and iOS
  OAuth clients in Google Cloud and add their IDs to the API's `GOOGLE_CLIENT_IDS`.

## Decided later

State management, offline support (`updated_at` on API rows allows `?updated_since=` sync),
push notifications, and in-app purchases vs web billing (3.13 / 5.01).
