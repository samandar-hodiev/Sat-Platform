import { ONBOARDING_ROUTE } from "@/features/onboarding/routes";

/** Profile Setup (PRD §7). Sign-up sends new learners here (AFTER_SIGN_UP in features/auth). */
export const PROFILE_SETUP_ROUTE = "/setup/profile";

/** Where Profile Setup continues. PRD §7: onboarding starts after it (session 2.02). */
export const AFTER_PROFILE_SETUP = ONBOARDING_ROUTE;
