import { describe, expect, it } from "vitest";

import {
  fieldMessage,
  latestBirthDate,
  phoneForApi,
  validateAvatarFile,
  validateDateOfBirth,
  validateName,
  validatePhone,
} from "./validation";

describe("validateName", () => {
  it("accepts names in any script with common punctuation", () => {
    for (const n of [
      "Ali",
      "  Anne   Marie ",
      "O'Neil",
      "Jean-Luc",
      "Мухаммад",
      "Oʻrinboy",
      "Sr.",
    ]) {
      expect(validateName(n, "first")).toBeUndefined();
    }
  });
  it("rejects empty, digits, symbols and long names", () => {
    expect(validateName("  ", "last")).toBe("Enter your last name.");
    expect(validateName("Ali2", "first")).toMatch(/letters/);
    expect(validateName("-Ali", "first")).toMatch(/letters/);
    expect(validateName("Ali 😀", "first")).toMatch(/letters/);
    expect(validateName("a".repeat(101), "first")).toMatch(/at most 100/);
  });
});

describe("phone", () => {
  it("treats the untouched prefix as no phone", () => {
    expect(phoneForApi("+998 ")).toBe("");
    expect(phoneForApi("+998")).toBe("");
    expect(validatePhone("+998 ")).toBeUndefined();
  });
  it("accepts international numbers as typed", () => {
    for (const p of ["+998 90 123-45-67", "00998901234567", "+1 (415) 555-0100"]) {
      expect(validatePhone(p)).toBeUndefined();
    }
  });
  it("rejects local or malformed numbers", () => {
    for (const p of ["901234567", "+0123456", "+998 90 abc", "+12"]) {
      expect(validatePhone(p)).toBeDefined();
    }
  });
});

describe("date of birth", () => {
  const today = new Date(2026, 9, 12); // 12 Oct 2026
  it("allows learners who are 13 today", () => {
    expect(latestBirthDate(today)).toBe("2013-10-12");
    expect(validateDateOfBirth("2013-10-12", today)).toBeUndefined();
  });
  it("rejects younger learners, future, old and invalid dates", () => {
    expect(validateDateOfBirth("2013-10-13", today)).toMatch(/at least 13/);
    expect(validateDateOfBirth("2026-10-13", today)).toMatch(/real date/);
    expect(validateDateOfBirth("1899-12-31", today)).toMatch(/real date/);
    expect(validateDateOfBirth("2008-02-30", today)).toBe("Enter a valid date.");
    expect(validateDateOfBirth("", today)).toBe("Enter your date of birth.");
  });
  it("handles 29 February", () => {
    expect(latestBirthDate(new Date(2028, 1, 29))).toBe("2015-02-28");
  });
});

describe("validateAvatarFile", () => {
  const file = (type: string, size: number) => new File([new Uint8Array(size)], "x", { type });
  it("allows JPG, PNG and WebP up to 5 MB", () => {
    expect(validateAvatarFile(file("image/png", 10))).toBeUndefined();
    expect(validateAvatarFile(file("image/webp", 5 * 1024 * 1024))).toBeUndefined();
  });
  it("rejects other types and larger files", () => {
    expect(validateAvatarFile(file("image/gif", 10))).toMatch(/JPG, PNG or WebP/);
    expect(validateAvatarFile(file("image/svg+xml", 10))).toMatch(/JPG, PNG or WebP/);
    expect(validateAvatarFile(file("image/jpeg", 5 * 1024 * 1024 + 1))).toMatch(/5 MB/);
  });
});

it("fieldMessage turns API fragments into sentences", () => {
  expect(fieldMessage("is required")).toBe("Is required.");
  expect(fieldMessage("Use at least 8 characters.")).toBe("Use at least 8 characters.");
  expect(fieldMessage(undefined)).toBeUndefined();
});
