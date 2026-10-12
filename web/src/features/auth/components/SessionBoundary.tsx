"use client";

import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useState, type ReactNode } from "react";

import { ErrorState, LoadingState } from "@/components/ui";

import { AUTH_ROUTES } from "../routes";
import { getSessionState, refreshSession, useSession } from "../session";

/**
 * Wraps pages that need a signed-in learner (learner app, exam runner). On first load it turns
 * the refresh cookie into an in-memory session; without a valid session it sends the learner
 * to sign-in and back here afterwards. proxy.ts already redirects when no session cookie exists.
 */
export function SessionBoundary({ children }: { children: ReactNode }) {
  const session = useSession();
  const router = useRouter();
  const pathname = usePathname();
  const [failed, setFailed] = useState(false);
  const [retrying, setRetrying] = useState(false);

  useEffect(() => {
    if (getSessionState().status !== "loading") return;
    refreshSession().catch(() => setFailed(true));
  }, []);

  const retry = useCallback(() => {
    setRetrying(true);
    refreshSession()
      .then(() => setFailed(false))
      .catch(() => setFailed(true))
      .finally(() => setRetrying(false));
  }, []);

  useEffect(() => {
    if (session.status === "signed-out") {
      router.replace(`${AUTH_ROUTES.signIn}?next=${encodeURIComponent(pathname)}`);
    }
  }, [session.status, pathname, router]);

  if (failed && session.status === "loading") {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState
          title="We couldn't restore your session"
          description="Check your connection and try again."
          onRetry={retry}
          retrying={retrying}
        />
      </div>
    );
  }
  if (session.status === "signed-out") {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <LoadingState label="Redirecting to sign in…" />
      </div>
    );
  }
  // While loading, the shell renders with its own loading states; data calls wait for the
  // session through withAuth().
  return <>{children}</>;
}
