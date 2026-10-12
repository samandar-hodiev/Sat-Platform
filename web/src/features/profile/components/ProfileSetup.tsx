"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { ErrorState, LoadingState } from "@/components/ui";
import { useSession } from "@/features/auth/session";

import { useLearnerProfile } from "../profileStore";
import { AFTER_PROFILE_SETUP } from "../routes";
import { ProfileSetupForm } from "./ProfileSetupForm";

/** The /setup/profile page body: loads the profile and shows the form until setup is done. */
export function ProfileSetup() {
  const router = useRouter();
  const session = useSession();
  const { state, reload } = useLearnerProfile();
  const completed = state.status === "ready" && state.profile.profile_completed;

  useEffect(() => {
    if (completed) router.replace(AFTER_PROFILE_SETUP);
  }, [completed, router]);

  if (state.status === "error") {
    return (
      <ErrorState
        title="We couldn't load your profile"
        description="Check your connection and try again."
        onRetry={reload}
      />
    );
  }
  if (state.status !== "ready" || session.status !== "signed-in" || completed) {
    return <LoadingState label={completed ? "Continuing…" : "Loading your profile…"} />;
  }
  return <ProfileSetupForm key={state.userId} userId={state.userId} profile={state.profile} />;
}
