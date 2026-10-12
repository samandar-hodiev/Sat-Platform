import type { Metadata } from "next";

import { OnboardingFlow } from "@/features/onboarding";

export const metadata: Metadata = { title: "Set your goals" };

export default function OnboardingPage() {
  return <OnboardingFlow />;
}
