// API contract types shared by every feature. They mirror api/internal/httpx and
// api/openapi/openapi.yaml; when the API contract changes, these change in the same commit.

export type ApiErrorCode =
  | "BAD_REQUEST"
  | "VALIDATION_ERROR"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "METHOD_NOT_ALLOWED"
  | "CONFLICT"
  | "PAYLOAD_TOO_LARGE"
  | "RATE_LIMITED"
  | "INTERNAL_ERROR"
  | "SERVICE_UNAVAILABLE";

export interface ApiErrorBody {
  code: ApiErrorCode;
  message: string;
  details?: {
    fields?: Record<string, string>;
    [key: string]: unknown;
  };
  request_id?: string;
}

export interface PageMeta {
  page: number;
  page_size: number;
  total: number;
}

export type ApiEnvelope<T> =
  | { success: true; data: T; meta?: PageMeta }
  | { success: false; error: ApiErrorBody; data?: unknown };

/** A paginated list as returned by the API client. */
export interface Page<T> {
  items: T;
  meta: PageMeta;
}
