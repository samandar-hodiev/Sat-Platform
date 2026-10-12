// Client-side checks give instant feedback only; the API validates everything again.
// Limits match api/internal/auth (password.go, email.go).

export const PASSWORD_MIN_LENGTH = 8;
export const PASSWORD_MAX_LENGTH = 128;
export const CODE_LENGTH = 6;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | undefined {
  const email = normalizeEmail(value);
  if (!email) return "Enter your email address.";
  if (email.length > 254 || !EMAIL_PATTERN.test(email)) return "Enter a valid email address.";
  return undefined;
}

export function validateNewPassword(value: string): string | undefined {
  // Counted in characters (code points), like the API.
  const length = [...value].length;
  if (length === 0) return "Enter a password.";
  if (length < PASSWORD_MIN_LENGTH) return `Use at least ${PASSWORD_MIN_LENGTH} characters.`;
  if (length > PASSWORD_MAX_LENGTH) return `Use at most ${PASSWORD_MAX_LENGTH} characters.`;
  return undefined;
}

/** Keeps digits only, so pasted codes like "123 456" or "123-456" work. */
export function sanitizeCode(value: string): string {
  return value.replace(/\D/g, "").slice(0, CODE_LENGTH);
}

/**
 * Only same-site paths are allowed as a post-sign-in destination, so a crafted
 * ?next=https://evil.example link cannot send the learner to another site.
 */
export function safeNextPath(value: string | null | undefined, fallback: string): string {
  if (!value || !value.startsWith("/") || value.startsWith("//") || value.includes("\\")) {
    return fallback;
  }
  if (value.startsWith("/auth/")) return fallback;
  return value;
}
