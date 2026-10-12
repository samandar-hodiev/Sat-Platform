import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api/client";

import { describeAuthError } from "./errors";

const apiError = (
  status: number,
  code: ConstructorParameters<typeof ApiError>[1]["code"],
  extra = {},
) => new ApiError(status, { code, message: "Server says", ...extra });

describe("describeAuthError", () => {
  it("returns field errors for validation failures", () => {
    const view = describeAuthError(
      apiError(422, "VALIDATION_ERROR", { details: { fields: { email: "Email is invalid." } } }),
    );
    expect(view).toEqual({ fields: { email: "Email is invalid." } });
  });

  it("shows the API message for client errors", () => {
    expect(describeAuthError(apiError(401, "UNAUTHORIZED")).message).toBe("Server says");
    expect(describeAuthError(apiError(400, "BAD_REQUEST")).message).toBe("Server says");
  });

  it("hides server faults and unknown errors", () => {
    for (const err of [
      apiError(500, "INTERNAL_ERROR"),
      apiError(200, "INVALID_RESPONSE"),
      new Error("boom"),
    ]) {
      expect(describeAuthError(err).message).toBe("Something went wrong. Please try again.");
    }
  });

  it("explains rate limits, network and missing service", () => {
    expect(describeAuthError(apiError(429, "RATE_LIMITED")).message).toMatch(/Too many attempts/);
    expect(describeAuthError(apiError(0, "NETWORK_ERROR")).message).toBe("Server says");
    expect(describeAuthError(apiError(404, "NOT_FOUND")).message).toMatch(/not available/);
  });
});
