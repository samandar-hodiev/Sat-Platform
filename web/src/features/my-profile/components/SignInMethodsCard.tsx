"use client";

import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";

import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Modal,
  useToast,
} from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import { loadGoogleIdentity } from "@/features/auth/components/GoogleButton";
import { GoogleLogo } from "@/features/auth/components/icons";
import { describeAuthError } from "@/features/auth/errors";

import { linkGoogle, unlinkGoogle, type SignInMethods } from "../api";
import { EMPTY_REAUTH, reauthBody, validateReauth, type ReauthValues } from "../reauth";
import { ReauthFields } from "./ReauthFields";

/** How the learner can sign in, and linking/unlinking Google (PRD §20 "login usullari"). */
export function SignInMethodsCard({
  methods,
  onChanged,
}: {
  methods: SignInMethods;
  onChanged: (next: SignInMethods) => void;
}) {
  const [dialog, setDialog] = useState<"link" | "unlink" | null>(null);
  const google = methods.google;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Sign-in methods</CardTitle>
        <CardDescription>Ways you can sign in to SAT Platform.</CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        <ul className="divide-y divide-border" aria-label="Sign-in methods">
          <MethodRow
            title="Email code"
            description={`A one-time code sent to ${methods.email}. Always available.`}
            status={<Badge variant="success">On</Badge>}
          />
          <MethodRow
            title="Password"
            description={
              methods.has_password ? "Sign in with your email and password." : "No password set."
            }
            status={
              methods.has_password ? <Badge variant="success">Set</Badge> : <Badge>Not set</Badge>
            }
          />
          <MethodRow
            title="Google"
            description={
              google
                ? `Linked${google.email ? ` to ${google.email}` : ""}.`
                : "Sign in with your Google account."
            }
            status={google ? <Badge variant="success">Linked</Badge> : <Badge>Not linked</Badge>}
            action={
              google ? (
                <Button variant="outline" size="sm" onClick={() => setDialog("unlink")}>
                  Unlink
                </Button>
              ) : (
                <Button variant="outline" size="sm" onClick={() => setDialog("link")}>
                  <GoogleLogo size={16} />
                  Link Google
                </Button>
              )
            }
          />
        </ul>
      </CardContent>
      {dialog === "link" ? (
        <LinkGoogleModal
          methods={methods}
          onClose={() => setDialog(null)}
          onDone={(next) => {
            setDialog(null);
            onChanged(next);
          }}
        />
      ) : null}
      {dialog === "unlink" ? (
        <UnlinkGoogleModal
          methods={methods}
          onClose={() => setDialog(null)}
          onDone={(next) => {
            setDialog(null);
            onChanged(next);
          }}
        />
      ) : null}
    </Card>
  );
}

function MethodRow({
  title,
  description,
  status,
  action,
}: {
  title: string;
  description: string;
  status: ReactNode;
  action?: ReactNode;
}) {
  return (
    <li className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
      <div className="flex min-w-0 flex-col gap-0.5">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-fg">{title}</span>
          {status}
        </div>
        <span className="text-sm break-words text-fg-muted">{description}</span>
      </div>
      {action}
    </li>
  );
}

/** Step 1 confirms it's the learner, step 2 shows Google's own button (branding rules). */
function LinkGoogleModal({
  methods,
  onClose,
  onDone,
}: {
  methods: SignInMethods;
  onClose: () => void;
  onDone: (next: SignInMethods) => void;
}) {
  const { toast } = useToast();
  const [step, setStep] = useState<"confirm" | "google">("confirm");
  const [reauth, setReauth] = useState<ReauthValues>(EMPTY_REAUTH);
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const [formError, setFormError] = useState<string>();
  const [busy, setBusy] = useState(false);

  function next(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const found = validateReauth(methods.reauth_method, reauth);
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;
    setStep("google");
  }

  async function onCredential(idToken: string) {
    setBusy(true);
    setFormError(undefined);
    try {
      const updated = await linkGoogle({
        id_token: idToken,
        ...reauthBody(methods.reauth_method, reauth),
      });
      toast({ variant: "success", title: "Google account linked" });
      onDone(updated);
    } catch (err) {
      const view = describeAuthError(err);
      setErrors(view.fields);
      setFormError(view.message);
      // A wrong password or code is fixed on the first step.
      if (view.fields.current_password || view.fields.code) setStep("confirm");
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} dismissible={!busy} title="Link a Google account">
      {step === "confirm" ? (
        <form
          noValidate
          onSubmit={next}
          className="flex flex-col gap-4"
          aria-label="Confirm it's you"
        >
          {formError ? <FormAlert>{formError}</FormAlert> : null}
          <p className="text-sm text-fg-muted">First, confirm it&apos;s you.</p>
          <ReauthFields
            method={methods.reauth_method}
            email={methods.email}
            values={reauth}
            onChange={setReauth}
            errors={errors}
          />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit">Continue</Button>
          </div>
        </form>
      ) : (
        <div className="flex flex-col gap-4">
          {formError ? <FormAlert>{formError}</FormAlert> : null}
          <p className="text-sm text-fg-muted">Choose the Google account to link.</p>
          <GoogleCredentialButton busy={busy} onCredential={onCredential} />
          <div className="flex justify-end">
            <Button type="button" variant="ghost" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}

/**
 * Google's button, returning the ID token instead of signing in. Without
 * NEXT_PUBLIC_GOOGLE_CLIENT_ID it says Google is not set up rather than pretending to link.
 */
function GoogleCredentialButton({
  busy,
  onCredential,
}: {
  busy: boolean;
  onCredential: (idToken: string) => void;
}) {
  const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;
  const container = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "failed">("loading");
  const latest = useRef(onCredential);
  useEffect(() => {
    latest.current = onCredential;
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
          callback: ({ credential }) => {
            if (credential) latest.current(credential);
          },
        });
        container.current.replaceChildren();
        gis.renderButton(container.current, {
          type: "standard",
          theme: document.documentElement.dataset.theme === "dark" ? "filled_black" : "outline",
          size: "large",
          text: "continue_with",
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
  }, [clientId]);

  if (!clientId || status === "failed") {
    return (
      <FormAlert>
        {clientId
          ? "Google could not load. Check your connection and try again."
          : "Linking a Google account is not set up yet."}
      </FormAlert>
    );
  }
  return (
    <div className="relative flex h-(--control-lg) w-full items-center justify-center">
      {status === "loading" || busy ? (
        <Button variant="outline" size="lg" fullWidth loading className="absolute inset-0" disabled>
          <GoogleLogo />
          Continue with Google
        </Button>
      ) : null}
      <div
        ref={container}
        data-testid="link-google"
        aria-busy={busy || undefined}
        className={busy ? "invisible w-full" : "flex w-full justify-center"}
      />
    </div>
  );
}

function UnlinkGoogleModal({
  methods,
  onClose,
  onDone,
}: {
  methods: SignInMethods;
  onClose: () => void;
  onDone: (next: SignInMethods) => void;
}) {
  const { toast } = useToast();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const google = methods.google;
  const sameEmail = google?.email === methods.email;

  async function unlink() {
    setBusy(true);
    setError(undefined);
    try {
      const next = await unlinkGoogle();
      toast({ variant: "success", title: "Google account unlinked" });
      onDone(next);
    } catch (err) {
      setError(describeAuthError(err).message ?? "Google could not be unlinked. Please try again.");
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      dismissible={!busy}
      title="Unlink Google?"
      description={`You can still sign in with a code sent to ${methods.email}${methods.has_password ? " or with your password" : ""}.`}
      footer={
        <>
          <Button variant="ghost" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" loading={busy} disabled={busy} onClick={unlink}>
            Unlink Google
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        {error ? <FormAlert>{error}</FormAlert> : null}
        {sameEmail ? (
          <p className="text-sm text-fg-muted">
            This Google account uses your account email, so choosing &quot;Continue with
            Google&quot; when signing in will link it again.
          </p>
        ) : null}
      </div>
    </Modal>
  );
}
