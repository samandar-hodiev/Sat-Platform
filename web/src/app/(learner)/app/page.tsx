import type { Metadata } from "next";

export const metadata: Metadata = { title: "Dashboard" };

// Dashboard Home blocks (PRD §10.4) are session 2.04's work; it replaces this page.
export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="text-2xl font-semibold tracking-tight text-fg">Dashboard</h1>
      <p className="text-sm text-fg-muted">Your study overview will appear here.</p>
    </div>
  );
}
