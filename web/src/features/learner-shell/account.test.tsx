// @vitest-environment jsdom
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ProfileState } from "@/features/profile";

import { useShellAccount } from "./account";

let profileState: ProfileState = { status: "idle" };
vi.mock("@/features/auth/session", () => ({
  useSession: () => ({ status: "signed-in", user: { id: "u1", email: "ali.v@example.com" } }),
  signOut: vi.fn(),
}));
vi.mock("@/features/profile", () => ({
  useLearnerProfile: () => ({ state: profileState, reload: vi.fn() }),
  avatarSrc: (p: string | null | undefined) => (p ? `http://api.test${p}` : null),
  displayName: (p: { first_name: string | null; last_name: string | null }) =>
    [p.first_name, p.last_name].filter(Boolean).join(" ") || null,
}));

describe("useShellAccount", () => {
  beforeEach(() => {
    profileState = { status: "idle" };
  });

  it("uses the email name while the profile loads", () => {
    const { result } = renderHook(() => useShellAccount());
    expect(result.current).toMatchObject({
      status: "ready",
      user: { name: "ali.v", email: "ali.v@example.com", avatarUrl: null },
    });
  });

  it("shows the profile name and avatar", () => {
    profileState = {
      status: "ready",
      userId: "u1",
      profile: {
        email: "ali.v@example.com",
        first_name: "Ali",
        last_name: "Valiyev",
        phone: null,
        date_of_birth: "2008-05-01",
        avatar_url: "/api/v1/media/avatars/u1/a.jpg",
        timezone: "UTC",
        has_password: true,
        password_required: false,
        profile_completed: true,
        profile_completed_at: "2026-10-12T00:00:00Z",
      },
    };
    const { result } = renderHook(() => useShellAccount());
    expect(result.current).toMatchObject({
      status: "ready",
      user: { name: "Ali Valiyev", avatarUrl: "http://api.test/api/v1/media/avatars/u1/a.jpg" },
    });
  });
});
