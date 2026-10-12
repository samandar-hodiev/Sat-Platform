import { describe, expect, it } from "vitest";

import {
  normalizeEmail,
  safeNextPath,
  sanitizeCode,
  validateEmail,
  validateNewPassword,
} from "./validation";

describe("validateEmail", () => {
  it.each(["a@example.com", "  Learner@Example.COM ", "first.last+sat@mail.uz"])(
    "accepts %s",
    (value) => expect(validateEmail(value)).toBeUndefined(),
  );
  it.each([
    "",
    "   ",
    "plain",
    "a@b",
    "a b@example.com",
    "@example.com",
    `${"a".repeat(250)}@x.uz`,
  ])("rejects %j", (value) => expect(validateEmail(value)).toBeDefined());
  it("normalizes", () => expect(normalizeEmail("  A@B.Com ")).toBe("a@b.com"));
});

describe("validateNewPassword", () => {
  it("enforces 8..128 characters, counted as characters", () => {
    expect(validateNewPassword("")).toBe("Enter a password.");
    expect(validateNewPassword("1234567")).toMatch(/at least 8/);
    expect(validateNewPassword("12345678")).toBeUndefined();
    expect(validateNewPassword("😀".repeat(8))).toBeUndefined();
    expect(validateNewPassword("a".repeat(129))).toMatch(/at most 128/);
  });
});

describe("sanitizeCode", () => {
  it("keeps up to six digits", () => {
    expect(sanitizeCode("123 456")).toBe("123456");
    expect(sanitizeCode("12-34-56-78")).toBe("123456");
    expect(sanitizeCode("ab1c")).toBe("1");
  });
});

describe("safeNextPath", () => {
  it.each([
    [null, "/app"],
    ["", "/app"],
    ["/app/practice?x=1", "/app/practice?x=1"],
    ["https://evil.example", "/app"],
    ["//evil.example", "/app"],
    ["/\\evil.example", "/app"],
    ["javascript:alert(1)", "/app"],
    ["/auth/sign-in", "/app"],
  ])("%j → %s", (value, expected) => expect(safeNextPath(value, "/app")).toBe(expected));
});
