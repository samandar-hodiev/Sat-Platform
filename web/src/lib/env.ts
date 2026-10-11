// Public runtime configuration. Only NEXT_PUBLIC_* values may be read here: they are
// bundled into client JavaScript. Secrets never belong in the web app.

export const API_VERSION = "v1";

/** Origin of the Go API, e.g. http://localhost:8090 (from NEXT_PUBLIC_API_URL). */
export function apiUrl(): string {
  // Next.js inlines NEXT_PUBLIC_* only when accessed literally like this.
  const value = process.env.NEXT_PUBLIC_API_URL;
  if (!value) {
    throw new Error("NEXT_PUBLIC_API_URL is not set. Copy web/.env.example to web/.env.local.");
  }
  return value.replace(/\/+$/, "");
}
