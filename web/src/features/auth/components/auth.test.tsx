// @vitest-environment jsdom
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ToastProvider } from "@/components/ui";
import { ApiError } from "@/lib/api/client";

import * as api from "../api";
import { ForgotPasswordForm } from "./ForgotPasswordForm";
import { ResetPasswordForm } from "./ResetPasswordForm";
import { SignInForm } from "./SignInForm";
import { SignUpFlow } from "./SignUpFlow";

const replace = vi.fn();
let search = new URLSearchParams();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace, push: vi.fn() }),
  useSearchParams: () => search,
}));
vi.mock("../api");

const mocked = vi.mocked(api);
const renderUi = (ui: ReactElement) => render(<ToastProvider>{ui}</ToastProvider>);
const session = {
  access_token: "access",
  access_token_expires_at: "2030-01-01T00:00:00Z",
  user: {
    id: "u1",
    email: "a@example.com",
    role: "learner" as const,
    email_verified: true,
    has_password: true,
    created_at: "2026-10-12T00:00:00Z",
  },
  is_new_user: false,
};

beforeEach(() => {
  vi.resetAllMocks();
  search = new URLSearchParams();
});
afterEach(() => vi.useRealTimers());

describe("SignInForm", () => {
  it("validates before calling the API", async () => {
    renderUi(<SignInForm />);
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getByText("Enter your email address.")).toBeTruthy();
    expect(screen.getByText("Enter your password.")).toBeTruthy();
    expect(mocked.signIn).not.toHaveBeenCalled();
  });

  it("signs in with the normalized email and goes to a safe ?next path", async () => {
    search = new URLSearchParams({ next: "/app/practice" });
    mocked.signIn.mockResolvedValue(session);
    renderUi(<SignInForm />);
    await userEvent.type(screen.getByLabelText("Email"), "  A@Example.com ");
    await userEvent.type(screen.getByLabelText("Password"), "secret-pass");
    await userEvent.click(screen.getByLabelText("Remember me"));
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect(mocked.signIn).toHaveBeenCalledWith({
      email: "a@example.com",
      password: "secret-pass",
      remember: false,
    });
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app/practice"));
  });

  it("ignores an external ?next", async () => {
    search = new URLSearchParams({ next: "https://evil.example" });
    mocked.signIn.mockResolvedValue(session);
    renderUi(<SignInForm />);
    await userEvent.type(screen.getByLabelText("Email"), "a@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "x");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app"));
  });

  it("shows the API error and keeps the form usable", async () => {
    mocked.signIn.mockRejectedValue(
      new ApiError(401, { code: "UNAUTHORIZED", message: "Email or password is incorrect." }),
    );
    renderUi(<SignInForm />);
    await userEvent.type(screen.getByLabelText("Email"), "a@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "wrong");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect((await screen.findByRole("alert")).textContent).toContain(
      "Email or password is incorrect.",
    );
    expect(screen.getByRole("button", { name: "Sign in" }).getAttribute("aria-busy")).toBeNull();
    expect(replace).not.toHaveBeenCalled();
  });

  it("toggles password visibility", async () => {
    renderUi(<SignInForm />);
    const input = screen.getByLabelText("Password");
    expect(input.getAttribute("type")).toBe("password");
    await userEvent.click(screen.getByRole("button", { name: "Show password" }));
    expect(input.getAttribute("type")).toBe("text");
  });

  it("says Google is not set up instead of pretending", async () => {
    renderUi(<SignInForm />);
    await userEvent.click(screen.getByRole("button", { name: /Continue with Google/ }));
    expect(await screen.findByText("Google sign-in is not set up yet")).toBeTruthy();
  });
});

describe("GoogleButton with a client ID", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    delete window.google;
  });

  function fakeGis() {
    let callback: ((r: { credential?: string }) => void) | undefined;
    const renderButton = vi.fn((parent: HTMLElement) => {
      const b = document.createElement("button");
      b.textContent = "Google";
      parent.appendChild(b);
    });
    window.google = {
      accounts: {
        id: {
          initialize: vi.fn((c: { callback: typeof callback }) => {
            callback = c.callback;
          }),
          renderButton,
        },
      },
    };
    return { renderButton, signIn: (credential: string) => callback?.({ credential }) };
  }

  it("renders Google's button and signs in with the ID token", async () => {
    vi.stubEnv("NEXT_PUBLIC_GOOGLE_CLIENT_ID", "web-client.apps.googleusercontent.com");
    search = new URLSearchParams({ next: "/app/progress" });
    const gis = fakeGis();
    mocked.signInWithGoogle.mockResolvedValue(session);
    renderUi(<SignInForm />);

    await waitFor(() => expect(gis.renderButton).toHaveBeenCalled());
    expect(screen.getByRole("button", { name: "Google" })).toBeTruthy();
    await act(async () => gis.signIn("id-token"));
    expect(mocked.signInWithGoogle).toHaveBeenCalledWith({ id_token: "id-token" });
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/app/progress"));
  });

  it("sends a new Google learner to Profile Setup", async () => {
    vi.stubEnv("NEXT_PUBLIC_GOOGLE_CLIENT_ID", "web-client");
    const gis = fakeGis();
    mocked.signInWithGoogle.mockResolvedValue({ ...session, is_new_user: true });
    renderUi(<SignUpFlow />);
    await waitFor(() => expect(gis.renderButton).toHaveBeenCalled());
    await act(async () => gis.signIn("id-token"));
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/setup/profile"));
  });

  it("shows the API error in a toast", async () => {
    vi.stubEnv("NEXT_PUBLIC_GOOGLE_CLIENT_ID", "web-client");
    const gis = fakeGis();
    mocked.signInWithGoogle.mockRejectedValue(
      new ApiError(401, {
        code: "UNAUTHORIZED",
        message: "Your Google account email is not verified.",
      }),
    );
    renderUi(<SignInForm />);
    await waitFor(() => expect(gis.renderButton).toHaveBeenCalled());
    await act(async () => gis.signIn("id-token"));
    expect(await screen.findByText("Your Google account email is not verified.")).toBeTruthy();
    expect(replace).not.toHaveBeenCalled();
  });
});

describe("SignUpFlow", () => {
  const sent = (resendInSeconds: number) => ({
    expires_at: new Date(Date.now() + 600_000).toISOString(),
    resend_available_at: new Date(Date.now() + resendInSeconds * 1000).toISOString(),
  });

  async function reachCodeStep(resendIn = 60) {
    mocked.startSignUp.mockResolvedValue(sent(resendIn));
    renderUi(<SignUpFlow />);
    await userEvent.type(screen.getByLabelText("Email"), "New@Example.com");
    await userEvent.click(screen.getByRole("button", { name: "Continue with email" }));
    await screen.findByText("Check your email");
  }

  it("sends a code, then verifies it automatically on the sixth digit", async () => {
    mocked.verifySignUp.mockResolvedValue({ ...session, is_new_user: true });
    await reachCodeStep();
    expect(mocked.startSignUp).toHaveBeenCalledWith({ email: "new@example.com" });
    expect(screen.getByText("new@example.com")).toBeTruthy();

    await userEvent.type(screen.getByLabelText("Verification code"), "12 34 56");
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/setup/profile"));
    expect(mocked.verifySignUp).toHaveBeenCalledTimes(1);
    expect(mocked.verifySignUp).toHaveBeenCalledWith({ email: "new@example.com", code: "123456" });
  });

  it("shows a wrong code error and clears the input", async () => {
    mocked.verifySignUp.mockRejectedValue(
      new ApiError(422, {
        code: "VALIDATION_ERROR",
        message: "Invalid input.",
        details: { fields: { code: "The code is incorrect. 4 attempts left." } },
      }),
    );
    await reachCodeStep();
    const input = screen.getByLabelText("Verification code");
    await userEvent.type(input, "000000");
    expect(await screen.findByText("The code is incorrect. 4 attempts left.")).toBeTruthy();
    expect((input as HTMLInputElement).value).toBe("");
    expect(replace).not.toHaveBeenCalled();
  });

  it("requires six digits before verifying", async () => {
    await reachCodeStep();
    await userEvent.type(screen.getByLabelText("Verification code"), "123");
    await userEvent.click(screen.getByRole("button", { name: "Verify and continue" }));
    expect(screen.getByText("Enter the 6-digit code.")).toBeTruthy();
    expect(mocked.verifySignUp).not.toHaveBeenCalled();
  });

  it("counts down before a resend is allowed", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    await reachCodeStep(2);
    expect(screen.getByText(/resend in \ds/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "resend the code" })).toBeNull();

    await act(() => vi.advanceTimersByTimeAsync(3000));
    mocked.startSignUp.mockResolvedValue(sent(60));
    await userEvent.click(await screen.findByRole("button", { name: "resend the code" }));
    expect(await screen.findByText("We sent you a new code.")).toBeTruthy();
    expect(mocked.startSignUp).toHaveBeenCalledTimes(2);
    expect(screen.getByText(/resend in \d+s/)).toBeTruthy();
  });

  it("can go back to change the email", async () => {
    await reachCodeStep();
    await userEvent.click(screen.getByRole("button", { name: /Use a different email/ }));
    expect(screen.getByText("Create your account")).toBeTruthy();
  });

  it("shows an error when the code cannot be sent", async () => {
    mocked.startSignUp.mockRejectedValue(new ApiError(429, { code: "RATE_LIMITED", message: "x" }));
    renderUi(<SignUpFlow />);
    await userEvent.type(screen.getByLabelText("Email"), "a@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Continue with email" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/Too many attempts/);
  });
});

describe("ForgotPasswordForm", () => {
  it("confirms without revealing whether the account exists", async () => {
    mocked.requestPasswordReset.mockResolvedValue(undefined);
    renderUi(<ForgotPasswordForm />);
    await userEvent.type(screen.getByLabelText("Email"), "A@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Send reset link" }));
    expect(mocked.requestPasswordReset).toHaveBeenCalledWith({ email: "a@example.com" });
    expect((await screen.findByText(/If an account exists for/)).textContent).toContain(
      "a@example.com",
    );
  });
});

describe("ResetPasswordForm", () => {
  it("shows an invalid-link state without a token", () => {
    renderUi(<ResetPasswordForm token={undefined} />);
    expect(screen.getByText("This link is not valid")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Request a new link" }).getAttribute("href")).toBe(
      "/auth/forgot-password",
    );
  });

  it("checks length and confirmation, then resets", async () => {
    mocked.resetPassword.mockResolvedValue(undefined);
    renderUi(<ResetPasswordForm token="tok" />);
    await userEvent.type(screen.getByLabelText("New password"), "short");
    await userEvent.click(screen.getByRole("button", { name: "Update password" }));
    expect(screen.getByText("Use at least 8 characters.")).toBeTruthy();

    await userEvent.type(screen.getByLabelText("New password"), "-enough");
    await userEvent.type(screen.getByLabelText("Confirm new password"), "different");
    await userEvent.click(screen.getByRole("button", { name: "Update password" }));
    expect(screen.getByText("Passwords do not match.")).toBeTruthy();
    expect(mocked.resetPassword).not.toHaveBeenCalled();

    await userEvent.clear(screen.getByLabelText("Confirm new password"));
    await userEvent.type(screen.getByLabelText("Confirm new password"), "short-enough");
    await userEvent.click(screen.getByRole("button", { name: "Update password" }));
    expect(mocked.resetPassword).toHaveBeenCalledWith({ token: "tok", password: "short-enough" });
    expect(await screen.findByText("Password updated")).toBeTruthy();
  });

  it("offers a new link when the token is rejected", async () => {
    mocked.resetPassword.mockRejectedValue(
      new ApiError(400, {
        code: "BAD_REQUEST",
        message: "This reset link is invalid or has expired.",
      }),
    );
    renderUi(<ResetPasswordForm token="old" />);
    await userEvent.type(screen.getByLabelText("New password"), "new-password");
    await userEvent.type(screen.getByLabelText("Confirm new password"), "new-password");
    await userEvent.click(screen.getByRole("button", { name: "Update password" }));
    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("This reset link is invalid or has expired.");
    expect(alert.querySelector("a")?.getAttribute("href")).toBe("/auth/forgot-password");
  });
});
