import { describe, expect, it } from "vitest";

import { SIDEBAR_COOKIE, parseSidebarState, sidebarCookie } from "./sidebarState";

describe("sidebar state cookie", () => {
  it.each([
    ["collapsed", "collapsed"],
    ["expanded", "expanded"],
    [undefined, "expanded"],
    [null, "expanded"],
    ["", "expanded"],
    ["COLLAPSED", "expanded"],
    ["collapsed; Path=/", "expanded"],
  ])("parses %j as %s", (raw, expected) => {
    expect(parseSidebarState(raw)).toBe(expected);
  });

  it("writes a long-lived, site-wide, lax cookie", () => {
    const cookie = sidebarCookie("collapsed");
    expect(cookie.startsWith(`${SIDEBAR_COOKIE}=collapsed;`)).toBe(true);
    expect(cookie).toContain("Path=/");
    expect(cookie).toMatch(/Max-Age=\d+/);
    expect(cookie).toContain("SameSite=Lax");
  });
});
