"use client";

import { useCallback, useEffect, useState } from "react";

import { ErrorState, Skeleton } from "@/components/ui";
import { useSession } from "@/features/auth/session";
import { useLearnerProfile } from "@/features/profile/profileStore";

import { fetchSignInMethods, type SignInMethods } from "../api";
import { DeleteAccountCard } from "./DeleteAccountCard";
import { EmailCard } from "./EmailCard";
import { PasswordCard } from "./PasswordCard";
import { ProfileDetailsCard } from "./ProfileDetailsCard";
import { SignInMethodsCard } from "./SignInMethodsCard";

type MethodsState =
  { status: "loading" } | { status: "error" } | { status: "ready"; methods: SignInMethods };

function useSignInMethods(userId: string | null) {
  const [state, setState] = useState<MethodsState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!userId) return;
    let cancelled = false;
    fetchSignInMethods().then(
      (methods) => !cancelled && setState({ status: "ready", methods }),
      () => !cancelled && setState({ status: "error" }),
    );
    return () => {
      cancelled = true;
    };
  }, [userId, attempt]);
  const reload = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((n) => n + 1);
  }, []);
  const set = useCallback((methods: SignInMethods) => setState({ status: "ready", methods }), []);
  return { state, reload, set };
}

/**
 * The /app/profile page body (PRD §20 Account): personal details and avatar, email, password,
 * sign-in methods and account deletion. The page sits inside the learner shell and ProfileGate,
 * so the profile is complete here.
 */
export function MyProfile() {
  const session = useSession();
  const userId = session.status === "signed-in" ? session.user.id : null;
  const { state: profile, reload: reloadProfile } = useLearnerProfile();
  const methods = useSignInMethods(userId);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight text-fg">My Profile</h1>
        <p className="text-sm text-fg-muted">Manage your personal details and how you sign in.</p>
      </div>

      {profile.status === "error" ? (
        <ErrorState
          title="We couldn't load your profile"
          description="Check your connection and try again."
          onRetry={reloadProfile}
        />
      ) : profile.status === "ready" ? (
        <ProfileDetailsCard
          key={`${profile.userId}:${profile.profile.profile_completed_at}`}
          userId={profile.userId}
          profile={profile.profile}
        />
      ) : (
        <Skeleton className="h-96" aria-label="Loading your profile" />
      )}

      {methods.state.status === "error" ? (
        <ErrorState
          title="We couldn't load your sign-in settings"
          description="Check your connection and try again."
          onRetry={methods.reload}
        />
      ) : methods.state.status === "ready" ? (
        <>
          <EmailCard methods={methods.state.methods} onChanged={methods.set} />
          <PasswordCard methods={methods.state.methods} onChanged={methods.reload} />
          <SignInMethodsCard methods={methods.state.methods} onChanged={methods.set} />
          <DeleteAccountCard methods={methods.state.methods} />
        </>
      ) : (
        <div className="flex flex-col gap-6" aria-label="Loading your sign-in settings">
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-56" />
        </div>
      )}
    </div>
  );
}
