import type { Metadata } from "next";

import { MyProfile } from "@/features/my-profile";

export const metadata: Metadata = { title: "My Profile" };

export default function MyProfilePage() {
  return <MyProfile />;
}
