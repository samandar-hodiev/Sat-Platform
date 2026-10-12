"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { ErrorState, LoadingState } from "@/components/ui";

import { useLearnerProfile } from "../profileStore";
import { PROFILE_SETUP_ROUTE } from "../routes";

/**
 * Learner app pages need a finished Profile Setup (PRD §7). Until the profile is known the page
 * area shows a loading state; an unfinished profile goes to /setup/profile. The shell around it
 * renders as usual.
 */
export function ProfileGate({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { state, reload } = useLearnerProfile();
  const incomplete = state.status === "ready" && !state.profile.profile_completed;

  useEffect(() => {
    if (incomplete) router.replace(PROFILE_SETUP_ROUTE);
  }, [incomplete, router]);

  if (state.status === "error") {
    return (
      <div className="flex justify-center p-6">
        <ErrorState
          title="We couldn't load your profile"
          description="Check your connection and try again."
          onRetry={reload}
        />
      </div>
    );
  }
  if (state.status !== "ready" || incomplete) {
    return (
      <div className="flex justify-center p-6">
        <LoadingState label={incomplete ? "Opening profile setup…" : "Loading…"} />
      </div>
    );
  }
  return <>{children}</>;
}
