# Authentication (architecture)

Status: **token model confirmed and implemented by 1.01** (JWT access + rotating opaque
refresh, as below). `api/internal/auth`: service, handlers, `Authenticate` middleware;
`web/src/features/auth`: session, `/api/session/*`, `proxy.ts` guard; PostgreSQL store and
migrations (docs/DB.md → Auth tables). Owner parts: 1.02. The design must work identically for the web app,
the Owner dashboard and the Android/iOS apps, against one API and one user table.

## Token model (recommended)

| Token | Format | Lifetime | Server side |
| --- | --- | --- | --- |
| Access token | Signed JWT (HS256): `sub` (user id), `role`, `sid` (session id), `iss`, `aud`, `exp`, `iat` | 15 minutes | Stateless; signature, issuer, audience and expiry are checked |
| Refresh token | 256-bit random, opaque | 30 days (owner: shorter, plus idle timeout) | SHA-256 hash in a `sessions`/`refresh_tokens` table with user, device, platform, created/last-used, revoked_at |

Every successful sign-in (email + password, email code, Google) returns the same JSON:

```json
{
  "access_token": "…", "access_token_expires_at": "2026-10-11T11:00:00Z",
  "refresh_token": "…", "refresh_token_expires_at": "2026-11-10T10:45:00Z",
  "token_type": "Bearer", "user": { "id": "…", "email": "…", "role": "learner" },
  "is_new_user": false
}
```

API requests authenticate with `Authorization: Bearer <access_token>` — the same for every
client. The Go API does not depend on cookies, so mobile apps work without a browser.

### Refresh and rotation

- `POST /api/v1/auth/refresh { refresh_token }` returns a new token pair and invalidates the
  presented refresh token (rotation).
- Reusing an already-rotated refresh token means it was stolen: the whole session is revoked
  and a security event is written to the audit log and notifications.
- Clients refresh single-flight (one refresh at a time), retry the failed request once, and
  sign the user out locally if refresh fails.

### Logout, revocation, multiple devices

- `POST /api/v1/auth/logout { refresh_token }` revokes that session server-side.
- Each sign-in creates its own session row, so a learner can be signed in on a laptop and a
  phone at once. "Active sessions" (list, revoke one, revoke all others) reads the same table.
- Password change/reset and account deletion revoke every session of the user.
- Access tokens are not revocable individually; the 15-minute lifetime bounds exposure.

## Sign-in methods (PRD §6)

| Method | Notes |
| --- | --- |
| Email + password | Argon2id hashes (parameters stored with the hash). Unknown email and wrong password give the same error. |
| Email verification code (sign-up) | Short numeric code, stored hashed, expires in minutes, limited attempts, resend cooldown and daily cap. Account is created after the code is verified. |
| Google | ID-token flow on every platform (below). No password needed; other methods can be linked later. |
| Password reset | Request always answers `202` (no account enumeration); single-use token stored hashed, ~1 hour. |
| Owner | Separate route `/owner/login` and API endpoints; owners are seeded/created by an admin process, never via registration; MFA (TOTP recommended); stricter rate limits; every login and important action audited. |

Every auth endpoint is rate-limited with its own scope on top of the global per-IP limit
(`middleware.RateLimit(limiter, "auth:login", …)`), keyed by IP and, where useful, by email.

### Google on web, Android and iOS

All clients obtain a Google **ID token** with the platform SDK and send it to
`POST /api/v1/auth/google { id_token }`:

| Client | SDK | Google Cloud OAuth client |
| --- | --- | --- |
| Web | Google Identity Services | Web client; authorized JavaScript origins = web origins |
| Android | `google_sign_in` / Credential Manager | Android client (package name + SHA-1) |
| iOS | `google_sign_in` | iOS client (bundle id) |

The API verifies the token signature against Google's keys, issuer, expiry,
`email_verified`, and that the audience is one of `GOOGLE_CLIENT_IDS` (all three client IDs).
No redirect URLs on the API and no client secret in any app. If a server-side redirect flow is
ever needed, it must use `state` + PKCE.

## Where clients keep tokens

### Web

| Token | Where |
| --- | --- |
| Access token | In memory only |
| Refresh token | `HttpOnly`, `Secure`, `SameSite=Strict` cookie scoped to a narrow path |

The cookie is set by a thin Next.js route handler (`/api/session/*`) that calls the Go API and
moves the refresh token into the cookie; the browser's JavaScript never sees it. CSRF defence:
`SameSite=Strict`, a narrow cookie path, and a required custom header (e.g.
`X-Session-Request: 1`) that cross-site forms cannot send. Calls to the Go API carry a Bearer
header, which browsers never attach automatically, so they are not CSRF-able. Nothing auth
related goes into `localStorage` (an ESLint rule forbids `localStorage` in the web app).

### Android and iOS (Flutter)

- Call `/api/v1/auth/*` directly. No cookies, no WebView session, no `localStorage`.
- Refresh token in `flutter_secure_storage` (iOS Keychain, Android Keystore-backed storage);
  access token in memory.
- Send `X-Client-Platform: android|ios`.
- On logout, call the API, then wipe secure storage.

## Authorization

- Roles are checked in the API on every request (`learner`, `owner`, and later staff roles if
  1.02 adds them). Owner routes use a `RequireRole/RequirePermission` middleware; the web
  hiding a page is cosmetic.
- Learner endpoints only touch the authenticated user's rows (user id from the token).
- Plan features (Premium) are checked in the API by 3.13, never only in a client.

### Web session as built (1.01)

| Piece | Where | Notes |
| --- | --- | --- |
| `sat_refresh` cookie | `/api/session/*` | Refresh token. HttpOnly, SameSite=Strict, Secure in production, `Path=/api/session`. `Max-Age` only with "Remember me" (otherwise a browser-session cookie; the server session is 24 h). |
| `sat_signed_in` cookie | `/` | Marker `1`, no secret, HttpOnly, SameSite=Lax. Read by `src/proxy.ts` to redirect `/app/*` and `/exam/*` to `/auth/sign-in?next=…` without a session. |
| Access token | memory (`features/auth/session.ts`) | `withAuth(call)` refreshes 30 s before expiry and retries once on 401. |
| Refresh | `refreshSession()` | Single-flight per tab. Two tabs racing within 30 s: the loser gets 401 and retries with the new cookie; the session survives. |
| Guard | `proxy.ts` + `SessionBoundary` | Proxy: fast redirect on missing marker. Boundary: restores the session on load, redirects when it has ended, offers retry on network errors. |

Rate limiting through the session handlers: the Go API sees the Next.js server as the client,
so the handler forwards the browser address in `X-Forwarded-For`. The API honours it only from
`TRUSTED_PROXIES`; production must list the web server there (5.05), otherwise every web user
shares one per-IP budget.

## Secrets and configuration (added by 1.01)

`JWT_SECRET` (32+ random chars, required in production), `AUTH_ACCESS_TOKEN_TTL`,
`AUTH_REFRESH_TOKEN_TTL`, `AUTH_SHORT_REFRESH_TOKEN_TTL`, `GOOGLE_CLIENT_IDS`, `MAIL_DRIVER`
(`log` | `disabled`). All live in the root `.env` read by the Go API only; none are
`NEXT_PUBLIC_*`. The web server may set `API_INTERNAL_URL` (server-only) when it reaches the API
on a private address.
