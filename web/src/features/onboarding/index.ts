// Onboarding (session 2.02). Contract: docs/API.md → Onboarding.
export {
  fetchOnboarding,
  type Onboarding,
  type OnboardingBackground,
  type OnboardingGoal,
  type OnboardingSchedule,
  type OnboardingStep,
  type PreviousScore,
  type SatReason,
} from "./api";
export { OnboardingFlow } from "./components/OnboardingFlow";
export { OnboardingGate } from "./components/OnboardingGate";
export { setOnboarding, useOnboarding, type OnboardingState } from "./onboardingStore";
export { AFTER_ONBOARDING, ONBOARDING_ROUTE } from "./routes";
