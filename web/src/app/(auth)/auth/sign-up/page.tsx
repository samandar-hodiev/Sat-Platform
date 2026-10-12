import type { Metadata } from "next";

import { SignUpFlow } from "@/features/auth/components/SignUpFlow";

export const metadata: Metadata = { title: "Create account" };

export default function SignUpPage() {
  return <SignUpFlow />;
}
