"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui";
import { AUTH_ROUTES } from "@/features/auth/routes";
import { signOut } from "@/features/auth/session";

/** "Not you? Sign out" for screens outside the learner shell (Profile Setup). */
export function SignOutButton() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      loading={busy}
      onClick={() => {
        setBusy(true);
        void signOut()
          .catch(() => undefined) // signed out locally either way
          .finally(() => router.replace(AUTH_ROUTES.signIn));
      }}
    >
      Sign out
    </Button>
  );
}
