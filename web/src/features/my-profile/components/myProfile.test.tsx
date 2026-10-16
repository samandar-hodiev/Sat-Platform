// @vitest-environment jsdom
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ToastProvider } from "@/components/ui";
import type { LearnerProfile } from "@/features/profile/api";
import { getProfileState, resetProfileForTests, setProfile } from "@/features/profile/profileStore";
import { ApiError } from "@/lib/api/client";

import * as api from "../api";
import type { SignInMethods } from "../api";
import { DeleteAccountCard } from "./DeleteAccountCard";
import { EmailCard } from "./EmailCard";
import { MyProfile } from "./MyProfile";
import { PasswordCard } from "./PasswordCard";
import { ProfileDetailsCard } from "./ProfileDetailsCard";
import { SignInMethodsCard } from "./SignInMethodsCard";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace, push: vi.fn() }) }));

const session = vi.hoisted(() => ({ signOut: vi.fn(), refreshSession: vi.fn() }));
vi.mock("@/features/auth/session", () => ({
  useSession: () => ({
    status: "signed-in",
    user: { id: "u1", email: "ali@example.com", role: "learner" },
  }),
  signOut: session.signOut,
  refreshSession: session.refreshSession,
  withAuth: vi.fn(),
}));
vi.mock("@/features/profile/api", () => ({
  fetchProfile: vi.fn(),
  uploadAvatar: vi.fn(),
  removeAvatar: vi.fn(),
  avatarSrc: (p: string | null) => (p ? `http://api.test${p}` : null),
}));
vi.mock("../api", () => ({
  updateProfile: vi.fn(),
  fetchSignInMethods: vi.fn(),
  sendReauthCode: vi.fn(),
  startEmailChange: vi.fn(),
  confirmEmailChange: vi.fn(),
  changePassword: vi.fn(),
  linkGoogle: vi.fn(),
  unlinkGoogle: vi.fn(),
  deleteAccount: vi.fn(),
}));
const mocked = vi.mocked(api);

function profile(over: Partial<LearnerProfile> = {}): LearnerProfile {
  return {
    email: "ali@example.com",
    first_name: "Ali",
    last_name: "Valiyev",
    phone: "+998901234567",
    date_of_birth: "2008-05-01",
    avatar_url: null,
    timezone: "Asia/Tashkent",
    has_password: true,
    password_required: false,
    profile_completed: true,
    profile_completed_at: "2026-10-12T09:00:00Z",
    ...over,
  };
}

function methods(over: Partial<SignInMethods> = {}): SignInMethods {
  return {
    email: "ali@example.com",
    email_verified: true,
    has_password: true,
    email_code: true,
    reauth_method: "password",
    google: null,
    ...over,
  };
}

function validationError(fields: Record<string, string>) {
  return new ApiError(422, {
    code: "VALIDATION_ERROR",
    message: "Some fields are invalid",
    details: { fields },
  });
}

function renderUI(ui: ReactElement) {
  return render(<ToastProvider>{ui}</ToastProvider>);
}

beforeEach(() => {
  vi.resetAllMocks();
  resetProfileForTests();
  session.refreshSession.mockResolvedValue(true);
  session.signOut.mockResolvedValue(undefined);
});

describe("ProfileDetailsCard", () => {
  it("saves only the changed fields and stores the result", async () => {
    const saved = profile({ first_name: "Aziza", phone: null });
    mocked.updateProfile.mockResolvedValue(saved);
    renderUI(<ProfileDetailsCard userId="u1" profile={profile()} />);
    const save = screen.getByRole("button", { name: "Save changes" });
    expect((save as HTMLButtonElement).disabled).toBe(true);

    const user = userEvent.setup();
    await user.clear(screen.getByLabelText(/First name/));
    await user.type(screen.getByLabelText(/First name/), "  Aziza ");
    await user.clear(screen.getByLabelText(/Phone number/));
    await user.click(save);

    await waitFor(() =>
      expect(mocked.updateProfile).toHaveBeenCalledWith({ first_name: "Aziza", phone: "" }),
    );
    expect(getProfileState()).toEqual({ status: "ready", userId: "u1", profile: saved });
    expect(await screen.findByText("Profile saved")).toBeTruthy();
  });

  it("validates before calling the API and can discard changes", async () => {
    renderUI(<ProfileDetailsCard userId="u1" profile={profile()} />);
    const user = userEvent.setup();
    await user.clear(screen.getByLabelText(/Last name/));
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(screen.getByText("Enter your last name.")).toBeTruthy();
    expect(mocked.updateProfile).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Discard changes" }));
    expect((screen.getByLabelText(/Last name/) as HTMLInputElement).value).toBe("Valiyev");
    expect(screen.queryByText("Enter your last name.")).toBeNull();
  });

  it("shows server field errors as sentences", async () => {
    mocked.updateProfile.mockRejectedValue(
      validationError({ phone: "must be an international number" }),
    );
    renderUI(<ProfileDetailsCard userId="u1" profile={profile()} />);
    const user = userEvent.setup();
    await user.clear(screen.getByLabelText(/Phone number/));
    await user.type(screen.getByLabelText(/Phone number/), "+998 90 000 00 00");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByText("Must be an international number.")).toBeTruthy();
  });
});

describe("EmailCard", () => {
  it("confirms with the password, then the code, and refreshes the session", async () => {
    mocked.startEmailChange.mockResolvedValue({
      new_email: "new@example.com",
      expires_at: "2026-10-17T10:10:00Z",
      resend_available_at: "2026-10-17T10:01:00Z",
    });
    const next = methods({ email: "new@example.com" });
    mocked.confirmEmailChange.mockResolvedValue(next);
    const onChanged = vi.fn();
    renderUI(<EmailCard methods={methods()} onChanged={onChanged} />);
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Change email" }));
    await user.click(screen.getByRole("button", { name: "Send code" }));
    expect(screen.getByText("Enter your current password.")).toBeTruthy();

    await user.type(screen.getByLabelText(/New email/), "ALI@example.com");
    await user.type(screen.getByLabelText(/Current password/), "secret-pass");
    await user.click(screen.getByRole("button", { name: "Send code" }));
    expect(screen.getByText("This is already your email.")).toBeTruthy();

    await user.clear(screen.getByLabelText(/New email/));
    await user.type(screen.getByLabelText(/New email/), " New@Example.com ");
    await user.click(screen.getByRole("button", { name: "Send code" }));
    await waitFor(() =>
      expect(mocked.startEmailChange).toHaveBeenCalledWith({
        new_email: "new@example.com",
        current_password: "secret-pass",
      }),
    );

    expect(await screen.findByText(/Enter the code we sent to new@example.com/)).toBeTruthy();
    await user.type(screen.getByLabelText(/^Code/), "12a3456");
    await user.click(screen.getByRole("button", { name: "Confirm email" }));
    await waitFor(() => expect(onChanged).toHaveBeenCalledWith(next));
    expect(mocked.confirmEmailChange).toHaveBeenCalledWith({ code: "123456" });
    expect(session.refreshSession).toHaveBeenCalled();
  });

  it("shows the server's reason on the right field", async () => {
    mocked.startEmailChange.mockRejectedValue(
      validationError({ new_email: "This email is already used by another account." }),
    );
    renderUI(<EmailCard methods={methods()} onChanged={vi.fn()} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Change email" }));
    await user.type(screen.getByLabelText(/New email/), "taken@example.com");
    await user.type(screen.getByLabelText(/Current password/), "secret-pass");
    await user.click(screen.getByRole("button", { name: "Send code" }));
    expect(await screen.findByText("This email is already used by another account.")).toBeTruthy();
  });
});

describe("PasswordCard", () => {
  it("sets a first password with an emailed code", async () => {
    mocked.sendReauthCode.mockResolvedValue({
      expires_at: new Date(Date.now() + 600_000).toISOString(),
      resend_available_at: new Date(Date.now() + 60_000).toISOString(),
    });
    mocked.changePassword.mockResolvedValue(undefined);
    const onChanged = vi.fn();
    renderUI(
      <PasswordCard
        methods={methods({ has_password: false, reauth_method: "email_code" })}
        onChanged={onChanged}
      />,
    );
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Set a password" }));
    const code = screen.getByLabelText(/Confirmation code/) as HTMLInputElement;
    expect(code.disabled).toBe(true); // until a code is sent

    await user.click(screen.getByRole("button", { name: "Email me a code" }));
    expect(await screen.findByRole("button", { name: /Resend code in \d+s/ })).toBeTruthy();
    expect(screen.getByText(/We sent a 6-digit code to ali@example.com/)).toBeTruthy();

    await user.type(code, "654321");
    await user.type(screen.getByLabelText(/^New password/), "brand-new-pass");
    await user.type(screen.getByLabelText(/Confirm new password/), "brand-new-pass");
    await user.click(screen.getByRole("button", { name: "Set password" }));
    await waitFor(() => expect(onChanged).toHaveBeenCalled());
    expect(mocked.changePassword).toHaveBeenCalledWith({
      new_password: "brand-new-pass",
      code: "654321",
    });
  });

  it("checks the new password locally and shows a wrong current password", async () => {
    mocked.changePassword.mockRejectedValue(
      validationError({ current_password: "This password is incorrect." }),
    );
    renderUI(<PasswordCard methods={methods()} onChanged={vi.fn()} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Change password" }));
    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText(/Current password/), "old-pass");
    await user.type(within(dialog).getByLabelText(/^New password/), "short");
    await user.type(within(dialog).getByLabelText(/Confirm new password/), "other");
    await user.click(within(dialog).getByRole("button", { name: "Change password" }));
    expect(within(dialog).getByText("Passwords do not match.")).toBeTruthy();
    expect(mocked.changePassword).not.toHaveBeenCalled();

    await user.clear(within(dialog).getByLabelText(/^New password/));
    await user.type(within(dialog).getByLabelText(/^New password/), "brand-new-pass");
    await user.clear(within(dialog).getByLabelText(/Confirm new password/));
    await user.type(within(dialog).getByLabelText(/Confirm new password/), "brand-new-pass");
    await user.click(within(dialog).getByRole("button", { name: "Change password" }));
    expect(await within(dialog).findByText("This password is incorrect.")).toBeTruthy();
  });
});

describe("SignInMethodsCard", () => {
  it("lists the methods and unlinks Google", async () => {
    const linked = methods({
      google: { email: "ali@gmail.com", linked_at: "2026-10-12T09:00:00Z" },
    });
    mocked.unlinkGoogle.mockResolvedValue(methods());
    const onChanged = vi.fn();
    renderUI(<SignInMethodsCard methods={linked} onChanged={onChanged} />);
    const list = screen.getByRole("list", { name: "Sign-in methods" });
    expect(within(list).getByText("Linked to ali@gmail.com.")).toBeTruthy();
    expect(within(list).getByText("Set")).toBeTruthy();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Unlink" }));
    await user.click(screen.getByRole("button", { name: "Unlink Google" }));
    await waitFor(() => expect(onChanged).toHaveBeenCalledWith(methods()));
  });

  it("does not pretend to link Google when it is not set up", async () => {
    renderUI(<SignInMethodsCard methods={methods()} onChanged={vi.fn()} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Link Google" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Enter your current password.")).toBeTruthy();
    await user.type(screen.getByLabelText(/Current password/), "secret-pass");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Linking a Google account is not set up yet.")).toBeTruthy();
    expect(mocked.linkGoogle).not.toHaveBeenCalled();
  });
});

describe("DeleteAccountCard", () => {
  it("needs DELETE and the password, then signs out and leaves", async () => {
    mocked.deleteAccount.mockResolvedValue(undefined);
    renderUI(<DeleteAccountCard methods={methods()} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Delete account" }));
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByText(/ali@example.com is freed/)).toBeTruthy();

    await user.type(within(dialog).getByLabelText(/Type DELETE/), "delete");
    await user.click(within(dialog).getByRole("button", { name: "Delete my account" }));
    expect(within(dialog).getByText("Type DELETE to confirm.")).toBeTruthy();
    expect(within(dialog).getByText("Enter your current password.")).toBeTruthy();
    expect(mocked.deleteAccount).not.toHaveBeenCalled();

    await user.clear(within(dialog).getByLabelText(/Type DELETE/));
    await user.type(within(dialog).getByLabelText(/Type DELETE/), "DELETE");
    await user.type(within(dialog).getByLabelText(/Current password/), "secret-pass");
    await user.click(within(dialog).getByRole("button", { name: "Delete my account" }));
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/"));
    expect(mocked.deleteAccount).toHaveBeenCalledWith({
      confirmation: "DELETE",
      current_password: "secret-pass",
    });
    expect(session.signOut).toHaveBeenCalled();
  });

  it("keeps the learner signed in when the server refuses", async () => {
    mocked.deleteAccount.mockRejectedValue(
      validationError({ current_password: "This password is incorrect." }),
    );
    renderUI(<DeleteAccountCard methods={methods()} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Delete account" }));
    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText(/Type DELETE/), "DELETE");
    await user.type(within(dialog).getByLabelText(/Current password/), "wrong");
    await user.click(within(dialog).getByRole("button", { name: "Delete my account" }));
    expect(await within(dialog).findByText("This password is incorrect.")).toBeTruthy();
    expect(session.signOut).not.toHaveBeenCalled();
    expect(replace).not.toHaveBeenCalled();
  });
});

describe("MyProfile", () => {
  it("shows loading, then every section", async () => {
    let resolve: (m: SignInMethods) => void = () => {};
    mocked.fetchSignInMethods.mockReturnValue(new Promise((r) => (resolve = r)));
    setProfile("u1", profile());
    renderUI(<MyProfile />);
    expect(screen.getByRole("heading", { name: "My Profile" })).toBeTruthy();
    expect(screen.getByLabelText("Loading your sign-in settings")).toBeTruthy();
    resolve(methods());
    expect(await screen.findByTestId("account-email")).toBeTruthy();
    for (const title of [
      "Personal details",
      "Email",
      "Password",
      "Sign-in methods",
      "Delete account",
    ]) {
      expect(screen.getAllByText(title).length).toBeGreaterThan(0);
    }
  });

  it("offers a retry when the sign-in settings fail to load", async () => {
    mocked.fetchSignInMethods.mockRejectedValueOnce(new Error("offline"));
    mocked.fetchSignInMethods.mockResolvedValueOnce(methods());
    setProfile("u1", profile());
    renderUI(<MyProfile />);
    expect(await screen.findByText("We couldn't load your sign-in settings")).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: /Try again/ }));
    expect(await screen.findByTestId("account-email")).toBeTruthy();
  });
});
