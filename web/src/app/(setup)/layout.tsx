import type { Metadata } from "next";
import Link from "next/link";

import { ThemeSwitcher } from "@/components/theme/ThemeSwitcher";
import { SessionBoundary } from "@/features/auth/components/SessionBoundary";
import { SignOutButton } from "@/features/profile/components/SignOutButton";

// Frame for the steps between sign-up and the learner app (session 2.01: Profile Setup; 2.02:
// onboarding). Signed-in learners only: proxy.ts and SessionBoundary (1.01) redirect otherwise.
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default function SetupLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <SessionBoundary>
      <div className="flex min-h-dvh flex-col bg-canvas">
        <header className="flex items-center justify-between gap-2 px-4 py-4 sm:px-6">
          <Link
            href="/"
            className="focus-ring flex items-center gap-2 rounded-md text-base font-semibold tracking-tight text-fg"
          >
            <span className="flex size-7 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-fg">
              S
            </span>
            SAT Platform
          </Link>
          <div className="flex items-center gap-1">
            <ThemeSwitcher />
            <SignOutButton />
          </div>
        </header>
        <main className="flex flex-1 justify-center px-4 pt-4 pb-12 sm:pt-8">
          <div className="w-full max-w-lg">{children}</div>
        </main>
      </div>
    </SessionBoundary>
  );
}
