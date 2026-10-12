// The single place the web app talks to the Go API. Features build typed functions on
// top of `api` (e.g. src/features/<feature>/api.ts) and never call fetch themselves.
//
// - Builds versioned URLs: ${NEXT_PUBLIC_API_URL}/api/v1${path}
// - Sends X-Client-Platform: web and, when given, Authorization: Bearer <token>
// - Unwraps { success, data, meta } and throws ApiError for every failure, including
//   network errors and non-JSON responses, so callers handle one error type.
// Token refresh is added by the auth session (1.01) around this client.

import { API_VERSION, apiUrl } from "@/lib/env";
import type { ApiEnvelope, ApiErrorBody, Page, PageMeta } from "@/types/api";

export type ClientErrorCode = ApiErrorBody["code"] | "NETWORK_ERROR" | "INVALID_RESPONSE";

export class ApiError extends Error {
  readonly status: number;
  readonly code: ClientErrorCode;
  readonly fieldErrors: Record<string, string>;
  readonly requestId: string | undefined;
  readonly details: ApiErrorBody["details"];

  constructor(
    status: number,
    body: Pick<ApiErrorBody, "message" | "details" | "request_id"> & { code: ClientErrorCode },
  ) {
    super(body.message);
    this.name = "ApiError";
    this.status = status;
    this.code = body.code;
    this.details = body.details;
    this.fieldErrors = body.details?.fields ?? {};
    this.requestId = body.request_id;
  }
}

type Query = Record<
  string,
  string | number | boolean | undefined | null | readonly (string | number)[]
>;

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  /** JSON-encoded, except FormData (file uploads), which is sent as multipart/form-data. */
  body?: unknown;
  query?: Query;
  accessToken?: string;
  signal?: AbortSignal;
  headers?: Record<string, string>;
}

export interface ClientConfig {
  baseUrl: string;
  version?: string;
  fetch?: typeof fetch;
}

export function buildUrl(baseUrl: string, version: string, path: string, query?: Query): string {
  const url = new URL(`${baseUrl}/api/${version}${path.startsWith("/") ? path : `/${path}`}`);
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const v of value) url.searchParams.append(key, String(v));
    } else {
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

export function createApiClient(config: ClientConfig) {
  const version = config.version ?? API_VERSION;
  const doFetch = config.fetch ?? fetch;

  async function send<T>(
    path: string,
    options: RequestOptions = {},
  ): Promise<{ data: T; meta?: PageMeta }> {
    const headers: Record<string, string> = {
      Accept: "application/json",
      "X-Client-Platform": "web",
      ...options.headers,
    };
    const multipart = typeof FormData !== "undefined" && options.body instanceof FormData;
    // For FormData the browser sets Content-Type itself, with the multipart boundary.
    if (options.body !== undefined && !multipart) headers["Content-Type"] = "application/json";
    if (options.accessToken) headers.Authorization = `Bearer ${options.accessToken}`;

    let response: Response;
    try {
      response = await doFetch(buildUrl(config.baseUrl, version, path, options.query), {
        method: options.method ?? "GET",
        headers,
        body:
          options.body === undefined
            ? undefined
            : multipart
              ? (options.body as FormData)
              : JSON.stringify(options.body),
        signal: options.signal,
      });
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") throw err;
      throw new ApiError(0, {
        code: "NETWORK_ERROR",
        message: "Could not reach the server. Check your connection.",
      });
    }

    if (response.status === 204) return { data: undefined as T };

    let envelope: ApiEnvelope<T>;
    try {
      envelope = (await response.json()) as ApiEnvelope<T>;
    } catch {
      throw new ApiError(response.status, {
        code: "INVALID_RESPONSE",
        message: "Unexpected response from the server.",
      });
    }

    if (!response.ok || !envelope.success) {
      const error = envelope.success ? undefined : envelope.error;
      throw new ApiError(response.status, {
        code: error?.code ?? "INVALID_RESPONSE",
        message: error?.message ?? "Unexpected response from the server.",
        details: error?.details,
        request_id: error?.request_id ?? response.headers.get("X-Request-ID") ?? undefined,
      });
    }
    return { data: envelope.data, meta: envelope.meta };
  }

  return {
    /** Request returning `data` only. */
    async request<T>(path: string, options?: RequestOptions): Promise<T> {
      return (await send<T>(path, options)).data;
    },
    /** Request for a paginated list: returns items plus `meta`. */
    async page<T>(path: string, options?: RequestOptions): Promise<Page<T>> {
      const { data, meta } = await send<T>(path, options);
      if (!meta) {
        throw new ApiError(200, {
          code: "INVALID_RESPONSE",
          message: "Expected a paginated response.",
        });
      }
      return { items: data, meta };
    },
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;

let shared: ApiClient | undefined;

/** The app-wide client, configured from NEXT_PUBLIC_API_URL on first use. */
export function api(): ApiClient {
  shared ??= createApiClient({ baseUrl: apiUrl() });
  return shared;
}
