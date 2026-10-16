// @vitest-environment jsdom
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider, useTheme } from "@/components/theme/ThemeProvider";
import { ToastProvider } from "@/components/ui";
import { ApiError } from "@/lib/api/client";
import { cookieName, readCookie } from "@/lib/theme/theme";

import { SIDEBAR_COOKIE } from "@/features/learner-shell";
import * as onboardingApi from "@/features/onboarding/api";

import * as api from "../api";
import type { Preferences } from "../api";
import { resetPreferencesForTests } from "../preferencesStore";
import { PreferencesSync } from "./PreferencesSync";
import { SettingsPage } from "./SettingsPage";

vi.mock("next/navigation", () => ({ usePathname: () => "/app/settings" }));
vi.mock("@/lib/env", () => ({ apiUrl: () => "http://api.test" }));
vi.mock("@/features/auth/session", () => ({
  useSession: () => ({
    status: "signed-in",
    user: { id: "u1", email: "ali@example.com", role: "learner" },
  }),
}));

vi.mock("../api", async (original) => ({
  ...(await original<typeof api>()),
  fetchPreferences: vi.fn(),
  updatePreferences: vi.fn(),
  uploadWallpaper: vi.fn(),
  removeWallpaper: vi.fn(),
}));
const mocked = vi.mocked(api);

const onboardingState = vi.hoisted(() => ({
  value: {
    completed: true,
    completed_at: "2026-10-12T10:00:00Z",
    next_step: null,
    goal: {
      target_score: 1400,
      exam_date: null,
      days_until_exam: null,
      reason: "scholarship",
      saved_at: "2026-10-12T10:00:00Z",
    },
    schedule: { daily_minutes: 60, study_days: [1, 3, 5], saved_at: "2026-10-12T10:00:00Z" },
    background: null,
  },
}));
const setOnboarding = vi.fn();
vi.mock("@/features/onboarding", () => ({
  useOnboarding: () => ({
    state: { status: "ready", userId: "u1", onboarding: onboardingState.value },
    reload: vi.fn(),
  }),
  setOnboarding: (...args: unknown[]) => setOnboarding(...args),
}));
vi.mock("@/features/onboarding/api", () => ({ saveGoal: vi.fn(), saveSchedule: vi.fn() }));
const mockedOnboarding = vi.mocked(onboardingApi);

function prefs(
  over: Partial<Preferences> = {},
  wallpaper: Partial<Preferences["wallpaper"]> = {},
): Preferences {
  return {
    theme: "light",
    accent: "neutral",
    density: "comfortable",
    motion: "system",
    sidebar_collapsed: false,
    stored: true,
    updated_at: "2026-10-17T00:00:00Z",
    ...over,
    wallpaper: {
      light: { source: "none", gallery_id: null, custom_url: null },
      dark: { source: "none", gallery_id: null, custom_url: null },
      opacity: 40,
      blur: 0,
      ...wallpaper,
    },
  };
}

const appearanceCookie = () => readCookie(document.cookie, cookieName("learner"));
const sidebarCookie = () => readCookie(document.cookie, SIDEBAR_COOKIE);

function AccentButton() {
  const { setAccent } = useTheme();
  return (
    <button type="button" onClick={() => setAccent("violet")}>
      violet
    </button>
  );
}

function renderWithProviders(ui: React.ReactNode) {
  return render(
    <ThemeProvider>
      <ToastProvider>{ui}</ToastProvider>
    </ThemeProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  resetPreferencesForTests();
});

describe("PreferencesSync", () => {
  it("applies the account's stored appearance to this device", async () => {
    document.cookie = `${cookieName("learner")}=light.blue.comfortable.system; Path=/`;
    mocked.fetchPreferences.mockResolvedValue(
      prefs({
        theme: "dark",
        accent: "rose",
        density: "compact",
        motion: "reduced",
        sidebar_collapsed: true,
      }),
    );
    renderWithProviders(<PreferencesSync initialSidebar="expanded" />);
    await waitFor(() => expect(appearanceCookie()).toBe("dark.rose.compact.reduced"));
    expect(sidebarCookie()).toBe("collapsed");
    expect(document.documentElement.dataset.theme).toBe("dark");
    // Applying the account values is not a change to save back.
    await new Promise((r) => setTimeout(r, 650));
    expect(mocked.updatePreferences).not.toHaveBeenCalled();
  });

  it("saves this device's choices to an account that never stored any", async () => {
    document.cookie = `${cookieName("learner")}=dark.teal.compact.full; Path=/`;
    mocked.fetchPreferences.mockResolvedValue(prefs({ stored: false, updated_at: null }));
    mocked.updatePreferences.mockResolvedValue(prefs({ theme: "dark", accent: "teal" }));
    renderWithProviders(<PreferencesSync initialSidebar="collapsed" />);
    await waitFor(() =>
      expect(mocked.updatePreferences).toHaveBeenCalledWith({
        theme: "dark",
        accent: "teal",
        density: "compact",
        motion: "full",
        sidebar_collapsed: true,
      }),
    );
    expect(appearanceCookie()).toBe("dark.teal.compact.full");
  });

  it("saves later changes once, after a pause", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.updatePreferences.mockResolvedValue(prefs({ accent: "violet" }));
    renderWithProviders(
      <>
        <PreferencesSync initialSidebar="expanded" />
        <AccentButton />
      </>,
    );
    await waitFor(() => expect(mocked.fetchPreferences).toHaveBeenCalled());
    await act(() => new Promise((r) => setTimeout(r, 0)));
    await userEvent.click(screen.getByRole("button", { name: "violet" }));
    await waitFor(() => expect(mocked.updatePreferences).toHaveBeenCalledTimes(1));
    expect(mocked.updatePreferences).toHaveBeenCalledWith(
      expect.objectContaining({ accent: "violet" }),
    );
  });

  it("keeps a change on this device and says so when saving fails", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.updatePreferences.mockRejectedValue(
      new ApiError(500, { code: "INTERNAL_ERROR", message: "x" }),
    );
    renderWithProviders(
      <>
        <PreferencesSync initialSidebar="expanded" />
        <AccentButton />
      </>,
    );
    await waitFor(() => expect(mocked.fetchPreferences).toHaveBeenCalled());
    await act(() => new Promise((r) => setTimeout(r, 0)));
    await userEvent.click(screen.getByRole("button", { name: "violet" }));
    expect(await screen.findByText("Appearance not saved to your account")).toBeTruthy();
    expect(appearanceCookie()).toBe("light.violet.comfortable.system");
  });
});

function group(name: RegExp) {
  return screen.getByRole("group", { name });
}

describe("Settings → Appearance", () => {
  it("changes theme, accent, density, motion and the sidebar at once", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(within(group(/^Theme/)).getByRole("radio", { name: /Dark/ }));
    await userEvent.click(within(group(/Accent color/)).getByRole("radio", { name: /Indigo/ }));
    await userEvent.click(within(group(/Density/)).getByRole("radio", { name: "Compact" }));
    await userEvent.click(within(group(/Animations/)).getByRole("radio", { name: "Reduced" }));
    expect(appearanceCookie()).toBe("dark.indigo.compact.reduced");
    expect(document.documentElement.dataset.accent).toBe("indigo");

    await userEvent.click(screen.getByRole("switch", { name: "Collapse the sidebar" }));
    expect(sidebarCookie()).toBe("collapsed");
  });

  it("chooses a gallery wallpaper per mode and saves it", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.updatePreferences.mockResolvedValue(
      prefs({}, { dark: { source: "gallery", gallery_id: "waves", custom_url: null } }),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(await screen.findByRole("radio", { name: "Dark mode" }));
    await userEvent.click(
      within(group(/Dark mode wallpaper/)).getByRole("radio", { name: "Waves" }),
    );
    expect(mocked.updatePreferences).toHaveBeenCalledWith({
      wallpaper: { dark: { source: "gallery", gallery_id: "waves" } },
    });
    expect(
      within(group(/Dark mode wallpaper/)).getByRole("radio", { name: "Waves" }),
    ).toHaveProperty("checked", true);
    // The preview is drawn in the edited mode.
    const preview = screen.getByTestId("wallpaper-preview");
    expect(preview.getAttribute("data-theme")).toBe("dark");
    within(preview).getByTestId("gallery-pattern-waves");
  });

  it("puts the choice back when saving fails", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.updatePreferences.mockRejectedValue(
      new ApiError(0, { code: "NETWORK_ERROR", message: "offline" }),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    const dots = within(
      await screen.findByRole("group", { name: /Light mode wallpaper/ }),
    ).getByRole("radio", {
      name: "Dots",
    });
    await userEvent.click(dots);
    expect(await screen.findByText("Wallpaper not saved")).toBeTruthy();
    expect(
      within(group(/Light mode wallpaper/)).getByRole("radio", { name: "None" }),
    ).toHaveProperty("checked", true);
  });

  it("checks the file before uploading and uploads a valid image for the mode", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.uploadWallpaper.mockResolvedValue(
      prefs(
        {},
        {
          light: {
            source: "custom",
            gallery_id: null,
            custom_url: "/api/v1/media/wallpapers/u1/a.jpg",
          },
        },
      ),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    const input = (await screen.findByTestId("wallpaper-file")) as HTMLInputElement;

    fireEvent.change(input, {
      target: { files: [new File(["GIF89a"], "a.gif", { type: "image/gif" })] },
    });
    expect(await screen.findByText("Upload a JPG, PNG or WebP image.")).toBeTruthy();
    const huge = new File(["x"], "big.png", { type: "image/png" });
    Object.defineProperty(huge, "size", { value: api.WALLPAPER_MAX_BYTES + 1 });
    fireEvent.change(input, { target: { files: [huge] } });
    expect(await screen.findByText("The image is larger than 8 MB.")).toBeTruthy();
    expect(mocked.uploadWallpaper).not.toHaveBeenCalled();

    const png = new File(["png"], "photo.png", { type: "image/png" });
    fireEvent.change(input, { target: { files: [png] } });
    await waitFor(() => expect(mocked.uploadWallpaper).toHaveBeenCalledWith("light", png));
    expect(await screen.findByRole("radio", { name: "Your image" })).toHaveProperty(
      "checked",
      true,
    );
    screen.getByRole("button", { name: "Replace your image" });
  });

  it("shows the server's reason when an upload is rejected", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mocked.uploadWallpaper.mockRejectedValue(
      new ApiError(422, {
        code: "VALIDATION_ERROR",
        message: "Some fields are invalid",
        details: { fields: { file: "This image can't be read. Try another file." } },
      }),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    const input = (await screen.findByTestId("wallpaper-file")) as HTMLInputElement;
    fireEvent.change(input, {
      target: { files: [new File(["x"], "a.jpg", { type: "image/jpeg" })] },
    });
    expect(await screen.findByText("This image can't be read. Try another file.")).toBeTruthy();
  });

  it("removes the uploaded image after confirmation", async () => {
    const custom = {
      source: "custom" as const,
      gallery_id: null,
      custom_url: "/api/v1/media/wallpapers/u1/a.jpg",
    };
    mocked.fetchPreferences.mockResolvedValue(prefs({}, { light: custom }));
    mocked.removeWallpaper.mockResolvedValue(prefs());
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(await screen.findByRole("button", { name: "Remove your image" }));
    const dialog = screen.getByRole("dialog", { name: "Remove your image?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Remove image" }));
    await waitFor(() => expect(mocked.removeWallpaper).toHaveBeenCalledWith("light"));
    expect(await screen.findByRole("button", { name: "Upload your image" })).toBeTruthy();
  });

  it("saves opacity and blur after the slider stops", async () => {
    mocked.fetchPreferences.mockResolvedValue(
      prefs({}, { light: { source: "gallery", gallery_id: "grid", custom_url: null } }),
    );
    mocked.updatePreferences.mockImplementation(async (patch) =>
      prefs({}, {
        light: { source: "gallery", gallery_id: "grid", custom_url: null },
        ...patch.wallpaper,
      } as never),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    const visibility = await screen.findByRole("slider", { name: "Visibility" });
    fireEvent.change(visibility, { target: { value: "55" } });
    fireEvent.change(visibility, { target: { value: "70" } });
    fireEvent.change(screen.getByRole("slider", { name: "Blur" }), { target: { value: "6" } });
    expect(screen.getByText("70%")).toBeTruthy();
    expect(mocked.updatePreferences).not.toHaveBeenCalled();
    await waitFor(() => expect(mocked.updatePreferences).toHaveBeenCalledTimes(1));
    expect(mocked.updatePreferences).toHaveBeenCalledWith({ wallpaper: { opacity: 70, blur: 6 } });
  });

  it("disables the sliders while neither mode has a wallpaper", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    expect(await screen.findByRole("slider", { name: "Visibility" })).toHaveProperty(
      "disabled",
      true,
    );
  });

  it("offers a retry when the preferences cannot be loaded", async () => {
    mocked.fetchPreferences.mockRejectedValue(
      new ApiError(0, { code: "NETWORK_ERROR", message: "offline" }),
    );
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    expect(await screen.findByText("Wallpaper settings could not be loaded")).toBeTruthy();
    screen.getByRole("button", { name: "Try again" });
  });
});

describe("Settings → Learning preferences and Notifications", () => {
  it("edits the goal and keeps the reason", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mockedOnboarding.saveGoal.mockResolvedValue(onboardingState.value as never);
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(screen.getByRole("tab", { name: "Learning preferences" }));
    const score = screen.getByRole("spinbutton", { name: /Target score/ });
    await userEvent.clear(score);
    await userEvent.type(score, "1405");
    await userEvent.click(screen.getByRole("button", { name: "Save goal" }));
    expect(await screen.findByText(/steps of 10|multiple of 10/i)).toBeTruthy();
    expect(mockedOnboarding.saveGoal).not.toHaveBeenCalled();

    await userEvent.clear(score);
    await userEvent.type(score, "1500");
    await userEvent.click(screen.getByRole("button", { name: "Save goal" }));
    await waitFor(() =>
      expect(mockedOnboarding.saveGoal).toHaveBeenCalledWith({
        target_score: 1500,
        exam_date: null,
        reason: "scholarship",
      }),
    );
    expect(setOnboarding).toHaveBeenCalledWith("u1", onboardingState.value);
    expect(await screen.findByText("Goal saved")).toBeTruthy();
  });

  it("edits the study time", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    mockedOnboarding.saveSchedule.mockResolvedValue(onboardingState.value as never);
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(screen.getByRole("tab", { name: "Learning preferences" }));
    const days = group(/Study days/);
    await userEvent.click(within(days).getByRole("checkbox", { name: "Monday" }));
    await userEvent.click(within(days).getByRole("checkbox", { name: "Sunday" }));
    await userEvent.click(screen.getByRole("button", { name: "Save study time" }));
    await waitFor(() =>
      expect(mockedOnboarding.saveSchedule).toHaveBeenCalledWith({
        daily_minutes: 60,
        study_days: [3, 5, 7],
      }),
    );
  });

  it("says notification settings are not available yet", async () => {
    mocked.fetchPreferences.mockResolvedValue(prefs());
    renderWithProviders(<SettingsPage initialSidebar="expanded" />);
    await userEvent.click(screen.getByRole("tab", { name: "Notifications" }));
    expect(screen.getByText("Notification settings are not available yet")).toBeTruthy();
    expect(screen.queryByRole("switch")).toBeNull();
  });
});
