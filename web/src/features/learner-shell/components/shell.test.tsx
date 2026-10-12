// @vitest-environment jsdom
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ToastProvider } from "@/components/ui";
import { readCookie } from "@/lib/theme/theme";

import type { ShellAccount } from "../account";
import { SIDEBAR_COOKIE } from "../sidebarState";
import { LearnerShell } from "./LearnerShell";

const nav = vi.hoisted(() => ({ pathname: "/app", push: vi.fn() }));
vi.mock("next/navigation", () => ({
  usePathname: () => nav.pathname,
  useRouter: () => ({ push: nav.push, replace: vi.fn(), prefetch: vi.fn() }),
}));

const session = vi.hoisted(() => ({ account: { status: "unavailable" } as ShellAccount }));
vi.mock("../account", () => ({ useShellAccount: () => session.account }));

function renderShell(initialSidebar: "expanded" | "collapsed" = "expanded") {
  return render(
    <ToastProvider>
      <LearnerShell initialSidebar={initialSidebar}>
        <h1>Page body</h1>
      </LearnerShell>
    </ToastProvider>,
  );
}

const sidebar = () => screen.getByTestId("learner-sidebar");
const drawer = () => screen.getByTestId("learner-drawer") as HTMLDialogElement;

beforeEach(() => {
  nav.pathname = "/app";
  nav.push.mockReset();
  session.account = { status: "unavailable" };
});

describe("LearnerShell layout", () => {
  it("renders landmarks, a skip link and the page content", () => {
    renderShell();
    expect(screen.getByRole("banner")).toBeTruthy();
    expect(screen.getByRole("main").id).toBe("main");
    expect(screen.getByRole("link", { name: "Skip to content" }).getAttribute("href")).toBe(
      "#main",
    );
    expect(screen.getByRole("heading", { name: "Page body" })).toBeTruthy();
  });

  it("shows the Learn, Track and Account groups as labelled lists", () => {
    renderShell();
    const main = within(sidebar()).getByRole("navigation", { name: "Main" });
    expect(within(main).getByRole("list", { name: "Learn" })).toBeTruthy();
    expect(within(main).getByRole("list", { name: "Track" })).toBeTruthy();
    expect(within(main).getByRole("list", { name: "Account" })).toBeTruthy();
    expect(within(main).getAllByRole("link")).toHaveLength(12);
  });

  it("marks the section of a nested page as current", () => {
    nav.pathname = "/app/exams/attempt-1/results";
    renderShell();
    const current = within(sidebar()).getByRole("link", { current: "page" });
    expect(current.textContent).toBe("Mock Exams");
    expect(
      within(sidebar()).getByRole("link", { name: "Dashboard" }).getAttribute("aria-current"),
    ).toBeNull();
  });

  it("shows the section in the breadcrumb, as a link on nested pages", () => {
    nav.pathname = "/app/exams";
    const { unmount } = renderShell();
    const crumb = within(screen.getByRole("navigation", { name: "Breadcrumb" }));
    expect(crumb.getByText("Mock Exams").getAttribute("aria-current")).toBe("page");
    unmount();

    nav.pathname = "/app/exams/attempt-1";
    renderShell();
    const nested = within(screen.getByRole("navigation", { name: "Breadcrumb" }));
    expect(nested.getByRole("link", { name: "Mock Exams" }).getAttribute("href")).toBe(
      "/app/exams",
    );
  });

  it("leaves the breadcrumb empty on an unknown page", () => {
    nav.pathname = "/app/nothing-here";
    renderShell();
    expect(screen.getByRole("navigation", { name: "Breadcrumb" }).textContent).toBe("");
  });
});

describe("collapsible sidebar", () => {
  it("collapses to icons, keeps accessible names and remembers the choice in a cookie", async () => {
    const user = userEvent.setup();
    renderShell();
    expect(sidebar().dataset.state).toBe("expanded");

    await user.click(screen.getByRole("button", { name: "Collapse sidebar" }));
    expect(sidebar().dataset.state).toBe("collapsed");
    expect(readCookie(document.cookie, SIDEBAR_COOKIE)).toBe("collapsed");
    const link = within(sidebar()).getByRole("link", { name: "Saved Questions" });
    expect(link.getAttribute("title")).toBe("Saved Questions");

    const expand = screen.getByRole("button", { name: "Expand sidebar" });
    expect(expand.getAttribute("aria-expanded")).toBe("false");
    await user.click(expand);
    expect(sidebar().dataset.state).toBe("expanded");
    expect(readCookie(document.cookie, SIDEBAR_COOKIE)).toBe("expanded");
  });

  it("starts collapsed when the server read a collapsed cookie", () => {
    renderShell("collapsed");
    expect(sidebar().dataset.state).toBe("collapsed");
    expect(within(sidebar()).getByRole("link", { name: "SAT Platform — Dashboard" })).toBeTruthy();
  });
});

describe("mobile drawer", () => {
  it("opens from the menu button and closes on Escape", async () => {
    const user = userEvent.setup();
    renderShell();
    const opener = screen.getByRole("button", { name: "Open navigation" });
    expect(drawer().open).toBe(false);
    expect(within(drawer()).queryByRole("navigation")).toBeNull();

    await user.click(opener);
    expect(drawer().open).toBe(true);
    expect(opener.getAttribute("aria-expanded")).toBe("true");
    expect(within(drawer()).getAllByRole("link").length).toBeGreaterThanOrEqual(12);

    act(() => {
      fireEvent(drawer(), new Event("cancel", { cancelable: true }));
    });
    expect(drawer().open).toBe(false);
    expect(opener.getAttribute("aria-expanded")).toBe("false");
  });

  it("closes after choosing a link and from the close button", async () => {
    const user = userEvent.setup();
    renderShell();
    await user.click(screen.getByRole("button", { name: "Open navigation" }));
    await user.click(within(drawer()).getByRole("link", { name: "My Progress" }));
    expect(drawer().open).toBe(false);

    await user.click(screen.getByRole("button", { name: "Open navigation" }));
    await user.click(within(drawer()).getByRole("button", { name: "Close navigation" }));
    expect(drawer().open).toBe(false);
  });
});

describe("profile menu", () => {
  it("shows a loading placeholder while the account loads", () => {
    session.account = { status: "loading" };
    renderShell();
    expect(screen.getByRole("status", { name: "Loading account" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Account menu" })).toBeNull();
  });

  it("does not invent a user when there is no session data", async () => {
    const user = userEvent.setup();
    renderShell();
    await user.click(screen.getByRole("button", { name: "Account menu" }));
    const menu = screen.getByRole("menu");
    expect(within(menu).getByText("Account details are unavailable")).toBeTruthy();
    expect(
      within(menu).getByRole("menuitem", { name: "Sign out" }).getAttribute("aria-disabled"),
    ).toBe("true");
  });

  it("navigates to account pages", async () => {
    const user = userEvent.setup();
    renderShell();
    await user.click(screen.getByRole("button", { name: "Account menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Settings" }));
    expect(nav.push).toHaveBeenCalledWith("/app/settings");
  });

  it("shows the signed-in learner and signs out", async () => {
    const signOut = vi.fn().mockResolvedValue(undefined);
    session.account = {
      status: "ready",
      user: { name: "Aziza Karimova", email: "aziza@example.com" },
      signOut,
    };
    const user = userEvent.setup();
    renderShell();
    expect(screen.getByRole("img", { name: "Aziza Karimova" }).textContent).toBe("AK");

    await user.click(screen.getByRole("button", { name: "Account menu" }));
    expect(screen.getByText("aziza@example.com")).toBeTruthy();
    await user.click(screen.getByRole("menuitem", { name: "Sign out" }));
    expect(signOut).toHaveBeenCalledTimes(1);
  });

  it("reports a failed sign-out", async () => {
    session.account = {
      status: "ready",
      user: { name: "Aziza Karimova", email: "aziza@example.com" },
      signOut: vi.fn().mockRejectedValue(new Error("network")),
    };
    const user = userEvent.setup();
    renderShell();
    await user.click(screen.getByRole("button", { name: "Account menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Sign out" }));
    await waitFor(() => expect(screen.getByText("Could not sign out")).toBeTruthy());
  });
});
