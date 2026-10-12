export const AUTH_ROUTES = {
  signIn: "/auth/sign-in",
  signUp: "/auth/sign-up",
  forgotPassword: "/auth/forgot-password",
  resetPassword: "/auth/reset-password",
  terms: "/terms", // page by 0.03
  privacy: "/privacy", // page by 0.03
} as const;

/** Where a signed-in learner lands by default (learner app, 2.03/2.04). */
export const AFTER_SIGN_IN = "/app";

/** Where a new learner goes after verifying the email: Profile Setup (PRD §7, session 2.01). */
export const AFTER_SIGN_UP = "/setup/profile";
