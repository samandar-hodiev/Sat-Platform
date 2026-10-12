import { ApiError } from "@/lib/api/client";

export interface AuthErrorView {
  /** Message for the form-level alert, if any. */
  message?: string;
  /** Messages for individual fields, keyed by API field name (email, password, code, …). */
  fields: Record<string, string>;
}

const GENERIC = "Something went wrong. Please try again.";
const UNAVAILABLE = "Sign-in is not available right now. Please try again later.";

/**
 * Turns any thrown value into what an auth form shows. API messages for 4xx errors are written to
 * be shown to users (api/internal/apperr), so they are used as is; server faults, unexpected
 * responses and unknown errors get a generic text and never leak details.
 */
export function describeAuthError(error: unknown): AuthErrorView {
  if (!(error instanceof ApiError)) return { message: GENERIC, fields: {} };

  switch (error.code) {
    case "VALIDATION_ERROR": {
      const fields = error.fieldErrors;
      return Object.keys(fields).length > 0
        ? { fields }
        : { message: error.message || GENERIC, fields: {} };
    }
    case "RATE_LIMITED":
      return {
        message: "Too many attempts. Please wait a moment and try again.",
        fields: {},
      };
    case "NETWORK_ERROR":
      return { message: error.message, fields: {} };
    case "NOT_FOUND":
    case "METHOD_NOT_ALLOWED":
    case "SERVICE_UNAVAILABLE":
      return { message: UNAVAILABLE, fields: {} };
    case "BAD_REQUEST":
    case "UNAUTHORIZED":
    case "FORBIDDEN":
    case "CONFLICT":
      return { message: error.message || GENERIC, fields: {} };
    default:
      return { message: GENERIC, fields: {} };
  }
}
