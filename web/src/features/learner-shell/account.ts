// Who is signed in, as the shell needs it (header avatar and profile menu).
//
// Backed by the web session of 1.01 (src/features/auth/session.ts) and the learner profile of
// 2.01 (src/features/profile). The name and picture come from the profile; while it loads, or if
// the learner has not entered a name, the part of the email before "@" is shown — real account
// data, not a made-up name.

import { signOut, useSession } from "@/features/auth/session";
import { avatarSrc, displayName, useLearnerProfile } from "@/features/profile";

export interface ShellUser {
  name: string;
  email: string;
  avatarUrl?: string | null;
}

export type ShellAccount =
  | { status: "loading" }
  | { status: "unavailable" }
  | { status: "ready"; user: ShellUser; signOut: () => Promise<void> };

export function useShellAccount(): ShellAccount {
  const session = useSession();
  const { state } = useLearnerProfile();
  if (session.status === "loading") return { status: "loading" };
  if (session.status === "signed-out") return { status: "unavailable" };
  const { email } = session.user;
  const profile = state.status === "ready" ? state.profile : null;
  return {
    status: "ready",
    user: {
      name: (profile && displayName(profile)) || email.split("@")[0] || email,
      email,
      avatarUrl: avatarSrc(profile?.avatar_url),
    },
    signOut,
  };
}
