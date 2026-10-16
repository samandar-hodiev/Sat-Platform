import { describe, expect, it } from "vitest";

import { reauthBody, secondsUntil, validateReauth } from "./reauth";

describe("reauth", () => {
  it("asks for the current password on password accounts", () => {
    expect(validateReauth("password", { password: "", code: "123456" })).toEqual({
      current_password: "Enter your current password.",
    });
    expect(validateReauth("password", { password: "x", code: "" })).toEqual({});
  });

  it("asks for a full code on accounts without a password", () => {
    expect(validateReauth("email_code", { password: "x", code: "" }).code).toMatch(
      /Enter the code/,
    );
    expect(validateReauth("email_code", { password: "", code: "123" }).code).toBe(
      "Enter the 6-digit code.",
    );
    expect(validateReauth("email_code", { password: "", code: "123456" })).toEqual({});
  });

  it("sends only the field that applies", () => {
    const values = { password: "secret-pass", code: "123456" };
    expect(reauthBody("password", values)).toEqual({ current_password: "secret-pass" });
    expect(reauthBody("email_code", values)).toEqual({ code: "123456" });
  });

  it("counts whole seconds down to zero", () => {
    const now = Date.parse("2026-10-17T10:00:00Z");
    expect(secondsUntil("2026-10-17T10:00:59.200Z", now)).toBe(60);
    expect(secondsUntil("2026-10-17T09:59:00Z", now)).toBe(0);
  });
});
