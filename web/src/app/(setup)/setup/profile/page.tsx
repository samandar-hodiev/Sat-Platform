import type { Metadata } from "next";

import { ProfileSetup } from "@/features/profile";

export const metadata: Metadata = { title: "Set up your profile" };

export default function ProfileSetupPage() {
  return <ProfileSetup />;
}
