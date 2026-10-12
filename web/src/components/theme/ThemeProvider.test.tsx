// @vitest-environment jsdom
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { cookieName, readCookie } from "@/lib/theme/theme";

import { ThemeProvider, useTheme } from "./ThemeProvider";
import { ThemeSwitcher } from "./ThemeSwitcher";

const pathname = vi.hoisted(() => ({ current: "/app" }));
vi.mock("next/navigation", () => ({ usePathname: () => pathname.current }));

function Probe() {
  const theme = useTheme();
  return (
    <>
      <output data-testid="state">{`${theme.scope}:${theme.mode}:${theme.resolvedTheme}:${theme.accent}`}</output>
      <button type="button" onClick={() => theme.setAccent("teal")}>
        teal
      </button>
    </>
  );
}

function renderTheme() {
  return render(
    <ThemeProvider>
      <ThemeSwitcher showLabels />
      <Probe />
    </ThemeProvider>,
  );
}

const html = () => document.documentElement;

describe("ThemeProvider", () => {
  beforeEach(() => {
    pathname.current = "/app";
    for (const name of ["data-theme", "data-accent", "data-density", "data-motion"]) {
      html().removeAttribute(name);
    }
  });

  it("starts in neutral light mode", () => {
    renderTheme();
    expect(screen.getByTestId("state").textContent).toBe("learner:light:light:neutral");
    expect(html().getAttribute("data-theme")).toBe("light");
    expect(screen.getByRole("radio", { name: "Light" }).getAttribute("aria-checked")).toBe("true");
  });

  it("applies and stores changes immediately", async () => {
    renderTheme();
    await userEvent.click(screen.getByRole("radio", { name: "Dark" }));
    await userEvent.click(screen.getByRole("button", { name: "teal" }));
    expect(html().getAttribute("data-theme")).toBe("dark");
    expect(html().getAttribute("data-accent")).toBe("teal");
    expect(readCookie(document.cookie, cookieName("learner"))).toBe("dark.teal.comfortable.system");
  });

  it("restores stored preferences", () => {
    document.cookie = `${cookieName("learner")}=dark.rose.compact.reduced; Path=/`;
    renderTheme();
    expect(screen.getByTestId("state").textContent).toBe("learner:dark:dark:rose");
    expect(html().getAttribute("data-density")).toBe("compact");
    expect(html().getAttribute("data-motion")).toBe("reduced");
  });

  it("follows the OS setting in system mode", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation(
      (query: string) =>
        ({
          matches: true,
          media: query,
          addEventListener: () => {},
          removeEventListener: () => {},
        }) as unknown as MediaQueryList,
    );
    renderTheme();
    await userEvent.click(screen.getByRole("radio", { name: "System" }));
    expect(screen.getByTestId("state").textContent).toBe("learner:system:dark:neutral");
    expect(html().getAttribute("data-theme")).toBe("dark");
    vi.restoreAllMocks();
  });

  it("keeps Owner preferences independent from Learner ones", async () => {
    document.cookie = `${cookieName("learner")}=dark.blue.comfortable.system; Path=/`;
    pathname.current = "/owner/learners";
    renderTheme();
    expect(screen.getByTestId("state").textContent).toBe("owner:light:light:neutral");
    await userEvent.click(screen.getByRole("radio", { name: "Dark" }));
    expect(readCookie(document.cookie, cookieName("owner"))).toBe(
      "dark.neutral.comfortable.system",
    );
    expect(readCookie(document.cookie, cookieName("learner"))).toBe("dark.blue.comfortable.system");
  });
});
