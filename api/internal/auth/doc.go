// Package auth implements learner authentication (session 1.01): email + password,
// email verification codes, Google sign-in, password reset and sessions.
//
// The token model is described in docs/AUTHENTICATION.md: a short-lived HS256 JWT access token
// and a rotating, opaque refresh token whose SHA-256 hash is stored per session. Every client
// (web, Android, iOS) uses the same endpoints and the same JSON.
//
// Secrets never appear in logs or errors: passwords, codes and tokens are only hashed,
// compared in constant time and returned to the client that asked for them.
package auth
