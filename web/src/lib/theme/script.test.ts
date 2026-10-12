// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from "vitest";

import { themeScript } from "./script";
import { cookieName, parsePreferences, themeAttributes, type ThemeScope } from "./theme";

function runScript() {
  new Function(themeScript())();
}

function attributes() {
  const html = document.documentElement;
  return Object.fromEntries(
    ["data-theme", "data-accent", "data-density", "data-motion"].map((name) => [
      name,
      html.getAttribute(name),
    ]),
  );
}

function setSystemDark(dark: boolean) {
  vi.spyOn(window, "matchMedia").mockImplementation(
    (query: string) => ({ matches: dark, media: query }) as MediaQueryList,
  );
}

describe("inline theme script", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    window.history.replaceState(null, "", "/");
  });

  const cases: Array<{ path: string; scope: ThemeScope; cookie: string | null; dark: boolean }> = [
    { path: "/", scope: "learner", cookie: null, dark: false },
    { path: "/", scope: "learner", cookie: null, dark: true },
    { path: "/app", scope: "learner", cookie: "dark.blue.compact.reduced", dark: false },
    { path: "/app", scope: "learner", cookie: "system.rose.comfortable.full", dark: true },
    { path: "/owner/learners", scope: "owner", cookie: "system.teal.compact.system", dark: false },
    { path: "/app", scope: "learner", cookie: "evil.<img>.x.y", dark: true },
  ];

  it.each(cases)("matches theme.ts for $path with $cookie (OS dark: $dark)", (testCase) => {
    window.history.replaceState(null, "", testCase.path);
    setSystemDark(testCase.dark);
    if (testCase.cookie) {
      document.cookie = `${cookieName(testCase.scope)}=${encodeURIComponent(testCase.cookie)}; Path=/`;
    }
    runScript();
    expect(attributes()).toEqual(themeAttributes(parsePreferences(testCase.cookie), testCase.dark));
  });

  it("ignores the other scope's cookie", () => {
    window.history.replaceState(null, "", "/owner");
    setSystemDark(false);
    document.cookie = `${cookieName("learner")}=dark.blue.compact.reduced; Path=/`;
    runScript();
    expect(attributes()["data-theme"]).toBe("light");
    expect(attributes()["data-accent"]).toBe("neutral");
  });
});
