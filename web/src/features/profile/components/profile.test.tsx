// @vitest-environment jsdom
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/lib/api/client";

import * as api from "../api";
import type { LearnerProfile } from "../api";
import { getProfileState, resetProfileForTests } from "../profileStore";
import { AvatarPicker } from "./AvatarPicker";
import { ProfileGate } from "./ProfileGate";
import { ProfileSetup } from "./ProfileSetup";
import { ProfileSetupForm } from "./ProfileSetupForm";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace, push: vi.fn() }) }));

const sessionUser = {
  id: "u1",
  email: "ali@example.com",
  role: "learner" as const,
  email_verified: true,
  has_password: false,
  created_at: "2026-10-12T00:00:00Z",
};
vi.mock("@/features/auth/session", () => ({
  useSession: () => ({ status: "signed-in", user: sessionUser }),
  signOut: vi.fn(),
}));
vi.mock("../api", () => ({
  fetchProfile: vi.fn(),
  completeProfileSetup: vi.fn(),
  uploadAvatar: vi.fn(),
  removeAvatar: vi.fn(),
  avatarSrc: (p: string | null) => (p ? `http://api.test${p}` : null),
  displayName: (p: LearnerProfile) => [p.first_name, p.last_name].filter(Boolean).join(" ") || null,
}));
const mocked = vi.mocked(api);

function profile(over: Partial<LearnerProfile> = {}): LearnerProfile {
  return {
    email: "ali@example.com",
    first_name: null,
    last_name: null,
    phone: null,
    date_of_birth: null,
    avatar_url: null,
    timezone: "UTC",
    has_password: false,
    password_required: true,
    profile_completed: false,
    profile_completed_at: null,
    ...over,
  };
}

beforeEach(() => {
  vi.resetAllMocks();
  resetProfileForTests();
});

async function fillRequired(user = userEvent.setup()) {
  await user.type(screen.getByLabelText(/First name/), "Ali");
  await user.type(screen.getByLabelText(/Last name/), "Valiyev");
  await user.type(screen.getByLabelText(/Date of birth/), "2008-05-01");
  return user;
}

describe("ProfileSetupForm", () => {
  it("validates before calling the API and requires a password for email accounts", async () => {
    render(<ProfileSetupForm userId="u1" profile={profile()} />);
    await userEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Enter your first name.")).toBeTruthy();
    expect(screen.getByText("Enter your last name.")).toBeTruthy();
    expect(screen.getByText("Enter your date of birth.")).toBeTruthy();
    expect(screen.getByText("Enter a password.")).toBeTruthy();
    expect(mocked.completeProfileSetup).not.toHaveBeenCalled();
  });

  it("shows the verified email read-only", () => {
    render(<ProfileSetupForm userId="u1" profile={profile()} />);
    const email = screen.getByLabelText("Email") as HTMLInputElement;
    expect(email.value).toBe("ali@example.com");
    expect(email.readOnly).toBe(true);
  });

  it("checks the password confirmation and the minimum age", async () => {
    render(<ProfileSetupForm userId="u1" profile={profile()} />);
    const user = userEvent.setup();
    await user.type(screen.getByLabelText(/Date of birth/), "2020-01-01");
    await user.type(screen.getByLabelText(/^Password/), "correct horse");
    await user.type(screen.getByLabelText(/Confirm password/), "different");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText(/at least 13 years old to use/)).toBeTruthy();
    expect(screen.getByText("Passwords do not match.")).toBeTruthy();
    expect(mocked.completeProfileSetup).not.toHaveBeenCalled();
  });

  it("submits the normalized form, stores the profile and continues", async () => {
    const saved = profile({ first_name: "Ali", profile_completed: true, has_password: true });
    mocked.completeProfileSetup.mockResolvedValue(saved);
    render(<ProfileSetupForm userId="u1" profile={profile()} />);
    const user = await fillRequired();
    await user.type(screen.getByLabelText(/First name/), "  ");
    await user.type(screen.getByLabelText(/^Password/), "correct horse");
    await user.type(screen.getByLabelText(/Confirm password/), "correct horse");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
    expect(mocked.completeProfileSetup).toHaveBeenCalledWith({
      first_name: "Ali",
      last_name: "Valiyev",
      phone: "", // untouched +998 prefix
      date_of_birth: "2008-05-01",
      timezone: expect.any(String),
      password: "correct horse",
    });
    expect(getProfileState()).toEqual({ status: "ready", userId: "u1", profile: saved });
  });

  it("prefills Google names and makes the password optional", async () => {
    mocked.completeProfileSetup.mockResolvedValue(profile({ profile_completed: true }));
    render(
      <ProfileSetupForm
        userId="u1"
        profile={profile({ first_name: "Ali", last_name: "Valiyev", password_required: false })}
      />,
    );
    expect((screen.getByLabelText(/First name/) as HTMLInputElement).value).toBe("Ali");
    expect(screen.getByText(/You can keep using Google/)).toBeTruthy();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText(/Date of birth/), "2008-05-01");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(mocked.completeProfileSetup).toHaveBeenCalled());
    expect(mocked.completeProfileSetup.mock.calls[0]?.[0].password).toBe("");
  });

  it("hides the password fields when the account has one", () => {
    render(
      <ProfileSetupForm
        userId="u1"
        profile={profile({ has_password: true, password_required: false })}
      />,
    );
    expect(screen.queryByLabelText(/^Password/)).toBeNull();
  });

  it("shows field errors from the API", async () => {
    mocked.completeProfileSetup.mockRejectedValue(
      new ApiError(422, {
        code: "VALIDATION_ERROR",
        message: "Some fields are invalid",
        details: { fields: { phone: "must be an international number, e.g. +998 90 123 45 67" } },
      }),
    );
    render(<ProfileSetupForm userId="u1" profile={profile({ password_required: false })} />);
    const user = await fillRequired();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText(/Must be an international number/)).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("continues when the profile was completed elsewhere", async () => {
    mocked.completeProfileSetup.mockRejectedValue(
      new ApiError(409, {
        code: "CONFLICT",
        message: "Your profile is already set up.",
        details: { reason: "profile_already_completed" },
      }),
    );
    render(<ProfileSetupForm userId="u1" profile={profile({ password_required: false })} />);
    const user = await fillRequired();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
  });

  it("shows a form error when the network fails", async () => {
    mocked.completeProfileSetup.mockRejectedValue(
      new ApiError(0, { code: "NETWORK_ERROR", message: "Could not reach the server." }),
    );
    render(<ProfileSetupForm userId="u1" profile={profile({ password_required: false })} />);
    const user = await fillRequired();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect((await screen.findByRole("alert")).textContent).toContain("Could not reach the server.");
  });
});

describe("AvatarPicker", () => {
  const png = () => new File([new Uint8Array(10)], "me.png", { type: "image/png" });

  it("shows the default avatar and uploads a chosen image", async () => {
    const onChange = vi.fn();
    const next = profile({ avatar_url: "/api/v1/media/avatars/u1/a.jpg" });
    mocked.uploadAvatar.mockResolvedValue(next);
    render(<AvatarPicker name="Ali Valiyev" avatarUrl={null} onChange={onChange} />);
    expect(screen.getByRole("img", { name: "Ali Valiyev" }).textContent).toBe("AV");
    await userEvent.upload(screen.getByTestId("avatar-input"), png());
    await waitFor(() => expect(onChange).toHaveBeenCalledWith(next));
  });

  it("rejects unsupported files without uploading", async () => {
    render(<AvatarPicker name="Ali" avatarUrl={null} onChange={vi.fn()} />);
    const gif = new File([new Uint8Array(10)], "x.gif", { type: "image/gif" });
    await userEvent.upload(screen.getByTestId("avatar-input"), gif, { applyAccept: false });
    expect(screen.getByRole("alert").textContent).toMatch(/JPG, PNG or WebP/);
    expect(mocked.uploadAvatar).not.toHaveBeenCalled();
  });

  it("shows the API's reason when an upload fails", async () => {
    mocked.uploadAvatar.mockRejectedValue(
      new ApiError(422, {
        code: "VALIDATION_ERROR",
        message: "Some fields are invalid",
        details: { fields: { file: "This image can't be read. Try another file." } },
      }),
    );
    render(<AvatarPicker name="Ali" avatarUrl={null} onChange={vi.fn()} />);
    await userEvent.upload(screen.getByTestId("avatar-input"), png());
    expect((await screen.findByRole("alert")).textContent).toBe(
      "This image can't be read. Try another file.",
    );
  });

  it("removes the photo", async () => {
    const onChange = vi.fn();
    mocked.removeAvatar.mockResolvedValue(profile());
    render(
      <AvatarPicker name="Ali" avatarUrl="/api/v1/media/avatars/u1/a.jpg" onChange={onChange} />,
    );
    expect(screen.getByRole("img", { name: "Ali" }).getAttribute("src")).toBe(
      "http://api.test/api/v1/media/avatars/u1/a.jpg",
    );
    await userEvent.click(screen.getByRole("button", { name: "Remove" }));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
  });
});

describe("ProfileGate", () => {
  it("sends learners without a finished profile to setup", async () => {
    mocked.fetchProfile.mockResolvedValue(profile());
    render(<ProfileGate>page</ProfileGate>);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/setup/profile"));
    expect(screen.queryByText("page")).toBeNull();
  });

  it("renders the page for a finished profile", async () => {
    mocked.fetchProfile.mockResolvedValue(profile({ profile_completed: true }));
    render(<ProfileGate>page</ProfileGate>);
    expect(await screen.findByText("page")).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });

  it("offers a retry when the profile cannot be loaded", async () => {
    mocked.fetchProfile
      .mockRejectedValueOnce(new ApiError(0, { code: "NETWORK_ERROR", message: "offline" }))
      .mockResolvedValueOnce(profile({ profile_completed: true }));
    render(<ProfileGate>page</ProfileGate>);
    await userEvent.click(await screen.findByRole("button", { name: /try again/i }));
    expect(await screen.findByText("page")).toBeTruthy();
    expect(mocked.fetchProfile).toHaveBeenCalledTimes(2);
  });
});

describe("ProfileSetup page", () => {
  it("shows the form for an unfinished profile", async () => {
    mocked.fetchProfile.mockResolvedValue(profile());
    render(<ProfileSetup />);
    expect(await screen.findByRole("heading", { name: "Set up your profile" })).toBeTruthy();
  });

  it("moves on when setup is already done", async () => {
    mocked.fetchProfile.mockResolvedValue(profile({ profile_completed: true }));
    render(<ProfileSetup />);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
  });
});
