import type { Metadata } from "next";
import { Suspense } from "react";

import { LoadingState } from "@/components/ui";
import { SignInForm } from "@/features/auth/components/SignInForm";

export const metadata: Metadata = { title: "Sign in" };

export default function SignInPage() {
  return (
    // SignInForm reads ?next= with useSearchParams, which needs a Suspense boundary.
    <Suspense fallback={<LoadingState />}>
      <SignInForm />
    </Suspense>
  );
}
