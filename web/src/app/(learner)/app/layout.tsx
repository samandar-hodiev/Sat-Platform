import type { Metadata } from "next";
import { cookies } from "next/headers";

import { SessionBoundary } from "@/features/auth/components/SessionBoundary";
import { LearnerShell, SIDEBAR_COOKIE, parseSidebarState } from "@/features/learner-shell";
import { OnboardingGate } from "@/features/onboarding";
import { ProfileGate } from "@/features/profile";
import { LearnerWallpaper, PreferencesSync } from "@/features/settings";

// Learner app layout (session 2.03, PRD §10.1–10.3). Every /app/* page renders inside it.
// Signed-in learners only: proxy.ts redirects without a session cookie, SessionBoundary (1.01)
// restores the session and redirects when it has ended. The API still checks every request.
// ProfileGate (2.01) sends learners who have not finished Profile Setup to /setup/profile, then
// OnboardingGate (2.02) sends those who have not finished onboarding to /setup/onboarding.
// PreferencesSync (2.11) keeps the appearance cookies and the account in step; LearnerWallpaper
// (2.11) draws the dashboard wallpaper behind the pages that show one.
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default async function LearnerLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const sidebar = parseSidebarState((await cookies()).get(SIDEBAR_COOKIE)?.value);
  return (
    <SessionBoundary>
      <PreferencesSync initialSidebar={sidebar} />
      <LearnerShell initialSidebar={sidebar} background={<LearnerWallpaper />}>
        <ProfileGate>
          <OnboardingGate>{children}</OnboardingGate>
        </ProfileGate>
      </LearnerShell>
    </SessionBoundary>
  );
}
