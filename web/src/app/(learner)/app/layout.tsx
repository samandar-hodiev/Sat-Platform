import type { Metadata } from "next";
import { cookies } from "next/headers";

import { SessionBoundary } from "@/features/auth/components/SessionBoundary";
import { LearnerShell, SIDEBAR_COOKIE, parseSidebarState } from "@/features/learner-shell";
import { OnboardingGate } from "@/features/onboarding";
import { ProfileGate } from "@/features/profile";

// Learner app layout (session 2.03, PRD §10.1–10.3). Every /app/* page renders inside it.
// Signed-in learners only: proxy.ts redirects without a session cookie, SessionBoundary (1.01)
// restores the session and redirects when it has ended. The API still checks every request.
// ProfileGate (2.01) sends learners who have not finished Profile Setup to /setup/profile, then
// OnboardingGate (2.02) sends those who have not finished onboarding to /setup/onboarding.
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default async function LearnerLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const sidebar = parseSidebarState((await cookies()).get(SIDEBAR_COOKIE)?.value);
  return (
    <SessionBoundary>
      <LearnerShell initialSidebar={sidebar}>
        <ProfileGate>
          <OnboardingGate>{children}</OnboardingGate>
        </ProfileGate>
      </LearnerShell>
    </SessionBoundary>
  );
}
