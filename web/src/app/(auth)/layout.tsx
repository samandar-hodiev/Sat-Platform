import Link from "next/link";

import { ThemeSwitcher } from "@/components/theme/ThemeSwitcher";

// Shared frame for learner auth screens (session 1.01): brand, theme switch, centered card.
export default function AuthLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <header className="flex items-center justify-between px-4 py-4 sm:px-6">
        <Link
          href="/"
          className="focus-ring flex items-center gap-2 rounded-md text-base font-semibold tracking-tight text-fg"
        >
          <span className="flex size-7 items-center justify-center rounded-md bg-primary text-sm font-bold text-primary-fg">
            S
          </span>
          SAT Platform
        </Link>
        <ThemeSwitcher />
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pt-6 pb-12 sm:items-center sm:pt-0">
        <div className="w-full max-w-sm">{children}</div>
      </main>
    </div>
  );
}
