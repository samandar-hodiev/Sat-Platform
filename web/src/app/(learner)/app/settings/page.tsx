import type { Metadata } from "next";
import { cookies } from "next/headers";

import { SIDEBAR_COOKIE, parseSidebarState } from "@/features/learner-shell";
import { SettingsPage } from "@/features/settings";

export const metadata: Metadata = { title: "Settings" };

export default async function LearnerSettingsPage() {
  const sidebar = parseSidebarState((await cookies()).get(SIDEBAR_COOKIE)?.value);
  return <SettingsPage initialSidebar={sidebar} />;
}
