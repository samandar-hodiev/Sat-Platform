import type { Metadata } from "next";

import { SessionBoundary } from "@/features/auth/components/SessionBoundary";

// Focus layout for the mock exam runner (session 3.08 builds /exam/<attemptId>): no sidebar,
// header or decorations, just the page on the canvas (PRD §10.1, §14). Wallpaper and other
// appearance effects from 2.11 must not apply here.
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default function FocusLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <SessionBoundary>
      <div data-layout="focus" className="flex min-h-dvh flex-col bg-canvas text-fg">
        <main id="main" tabIndex={-1} className="flex flex-1 flex-col outline-none">
          {children}
        </main>
      </div>
    </SessionBoundary>
  );
}
