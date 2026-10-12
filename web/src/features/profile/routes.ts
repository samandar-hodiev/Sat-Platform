/** Profile Setup (PRD §7). Sign-up sends new learners here (AFTER_SIGN_UP in features/auth). */
export const PROFILE_SETUP_ROUTE = "/setup/profile";

/**
 * Where Profile Setup continues. PRD §7: onboarding starts after it; the onboarding page is
 * session 2.02, which points this at its route. Until then the learner goes to the dashboard.
 */
export const AFTER_PROFILE_SETUP = "/app";
