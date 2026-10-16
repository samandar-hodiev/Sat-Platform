"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Button, useToast } from "@/components/ui";

import { signInWithGoogle } from "../api";
import { describeAuthError } from "../errors";
import { AFTER_SIGN_IN, AFTER_SIGN_UP } from "../routes";
import { GoogleLogo } from "./icons";

// Google Identity Services (https://developers.google.com/identity/gsi/web). The browser gets
// a Google ID token and the API verifies it (POST /auth/google via /api/session/google); no
// redirect URLs and no client secret on the web.
const GIS_SRC = "https://accounts.google.com/gsi/client";

export interface GoogleIdApi {
  initialize(config: {
    client_id: string;
    callback: (response: { credential?: string }) => void;
    ux_mode?: "popup";
    use_fedcm_for_button?: boolean;
  }): void;
  renderButton(
    parent: HTMLElement,
    options: {
      type: "standard";
      theme: "outline" | "filled_black";
      size: "large";
      text: "continue_with" | "signup_with";
      shape: "rectangular";
      logo_alignment: "center";
      width: number;
    },
  ): void;
}

declare global {
  interface Window {
    google?: { accounts?: { id?: GoogleIdApi } };
  }
}

let scriptPromise: Promise<GoogleIdApi> | null = null;

/** Loads Google Identity Services once; also used to link a Google account (My Profile, 2.10). */
export function loadGoogleIdentity(): Promise<GoogleIdApi> {
  const ready = window.google?.accounts?.id;
  if (ready) return Promise.resolve(ready);
  scriptPromise ??= new Promise<GoogleIdApi>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = GIS_SRC;
    script.async = true;
    script.onload = () => {
      const api = window.google?.accounts?.id;
      if (api) resolve(api);
      else reject(new Error("Google Identity Services did not load"));
    };
    script.onerror = () => {
      scriptPromise = null;
      reject(new Error("Google Identity Services did not load"));
    };
    document.head.appendChild(script);
  });
  return scriptPromise;
}

/**
 * "Continue with Google". With NEXT_PUBLIC_GOOGLE_CLIENT_ID set it renders Google's own button
 * (required by Google's branding rules); without it the button says Google is not set up
 * instead of pretending to sign in.
 */
export function GoogleButton({
  mode = "sign-in",
  next,
}: {
  mode?: "sign-in" | "sign-up";
  /** Where to go after signing in an existing learner (already validated by the caller). */
  next?: string;
}) {
  const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;
  const label = mode === "sign-up" ? "Sign up with Google" : "Continue with Google";
  const { toast } = useToast();
  const router = useRouter();
  const container = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "failed">("loading");
  const [busy, setBusy] = useState(false);
  // The GIS callback is registered once; it reads the latest navigation inputs from a ref.
  const latest = useRef({ router, toast, next });
  useEffect(() => {
    latest.current = { router, toast, next };
  });

  useEffect(() => {
    if (!clientId) return;
    let cancelled = false;
    loadGoogleIdentity()
      .then((gis) => {
        if (cancelled || !container.current) return;
        gis.initialize({
          client_id: clientId,
          ux_mode: "popup",
          use_fedcm_for_button: true,
          callback: async ({ credential }) => {
            if (!credential) return;
            const { router, toast, next } = latest.current;
            setBusy(true);
            try {
              const session = await signInWithGoogle({ id_token: credential });
              router.replace(session.is_new_user ? AFTER_SIGN_UP : (next ?? AFTER_SIGN_IN));
            } catch (err) {
              toast({
                variant: "error",
                title: "Google sign-in failed",
                description: describeAuthError(err).message,
              });
              setBusy(false);
            }
          },
        });
        const dark = document.documentElement.dataset.theme === "dark";
        // Effects can run twice (React strict mode, remounts): never stack two Google buttons.
        container.current.replaceChildren();
        gis.renderButton(container.current, {
          type: "standard",
          theme: dark ? "filled_black" : "outline",
          size: "large",
          text: mode === "sign-up" ? "signup_with" : "continue_with",
          shape: "rectangular",
          logo_alignment: "center",
          width: Math.min(400, Math.max(200, container.current.clientWidth)),
        });
        setStatus("ready");
      })
      .catch(() => !cancelled && setStatus("failed"));
    return () => {
      cancelled = true;
    };
  }, [clientId, mode]);

  if (!clientId || status === "failed") {
    return (
      <Button
        variant="outline"
        size="lg"
        fullWidth
        data-testid="auth-google"
        onClick={() =>
          toast({
            variant: "info",
            title: clientId ? "Google sign-in could not load" : "Google sign-in is not set up yet",
            description: clientId
              ? "Check your connection or use your email."
              : "Use your email for now.",
          })
        }
      >
        <GoogleLogo />
        {label}
      </Button>
    );
  }

  return (
    <div className="relative flex h-(--control-lg) w-full items-center justify-center">
      {status === "loading" || busy ? (
        <Button variant="outline" size="lg" fullWidth loading className="absolute inset-0" disabled>
          <GoogleLogo />
          {label}
        </Button>
      ) : null}
      <div
        ref={container}
        data-testid="auth-google"
        aria-busy={busy || undefined}
        className={busy ? "invisible w-full" : "flex w-full justify-center"}
      />
    </div>
  );
}

export function OrDivider() {
  return (
    <div className="flex items-center gap-3 text-xs text-fg-subtle uppercase" role="separator">
      <span className="h-px flex-1 bg-border" />
      or
      <span className="h-px flex-1 bg-border" />
    </div>
  );
}
