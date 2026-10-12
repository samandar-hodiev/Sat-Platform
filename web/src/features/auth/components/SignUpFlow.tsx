"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";

import { Button, FormField, Input } from "@/components/ui";

import { startSignUp, verifySignUp, type CodeSent } from "../api";
import { describeAuthError } from "../errors";
import { AFTER_SIGN_IN, AFTER_SIGN_UP, AUTH_ROUTES } from "../routes";
import { CODE_LENGTH, normalizeEmail, sanitizeCode, validateEmail } from "../validation";
import { AuthCard } from "./AuthCard";
import { FormAlert } from "./FormAlert";
import { GoogleButton, OrDivider } from "./GoogleButton";
import { ArrowLeftIcon, MailIcon } from "./icons";
import { LegalNote } from "./LegalNote";

/**
 * Sign-up by email (PRD §6.1): email → verification code → account created → Profile Setup.
 * The password is chosen in Profile Setup (PRD §7, session 2.01), not here.
 */
export function SignUpFlow() {
  const [step, setStep] = useState<{ email: string; sent: CodeSent } | null>(null);

  if (step) {
    return (
      <CodeStep
        email={step.email}
        sent={step.sent}
        onResent={(sent) => setStep({ email: step.email, sent })}
        onChangeEmail={() => setStep(null)}
      />
    );
  }
  return <EmailStep onSent={(email, sent) => setStep({ email, sent })} />;
}

function EmailStep({ onSent }: { onSent: (email: string, sent: CodeSent) => void }) {
  const [email, setEmail] = useState("");
  const [emailError, setEmailError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    const error = validateEmail(email);
    setEmailError(error);
    setFormError(undefined);
    if (error) return;

    setSubmitting(true);
    const normalized = normalizeEmail(email);
    try {
      const sent = await startSignUp({ email: normalized });
      onSent(normalized, sent);
    } catch (err) {
      const view = describeAuthError(err);
      setEmailError(view.fields.email);
      setFormError(view.message);
      setSubmitting(false);
    }
  }

  return (
    <AuthCard
      title="Create your account"
      description="Start preparing for the Digital SAT."
      footer={
        <>
          <p>
            Already have an account?{" "}
            <Link
              href={AUTH_ROUTES.signIn}
              className="font-medium text-fg underline-offset-4 hover:underline"
            >
              Sign in
            </Link>
          </p>
          <LegalNote />
        </>
      }
    >
      <GoogleButton mode="sign-up" />
      <OrDivider />
      <form noValidate onSubmit={onSubmit} className="flex flex-col gap-4" aria-label="Sign up">
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <FormField label="Email" hint="We'll send a 6-digit code to confirm it." error={emailError}>
          <Input
            type="email"
            name="email"
            autoComplete="email"
            inputMode="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            data-testid="sign-up-email"
          />
        </FormField>
        <Button type="submit" size="lg" fullWidth loading={submitting} data-testid="sign-up-submit">
          Continue with email
        </Button>
      </form>
    </AuthCard>
  );
}

function secondsUntil(iso: string, now: number): number {
  const at = Date.parse(iso);
  return Number.isNaN(at) ? 0 : Math.max(0, Math.ceil((at - now) / 1000));
}

function CodeStep({
  email,
  sent,
  onResent,
  onChangeEmail,
}: {
  email: string;
  sent: CodeSent;
  onResent: (sent: CodeSent) => void;
  onChangeEmail: () => void;
}) {
  const router = useRouter();
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [verifying, setVerifying] = useState(false);
  const [resending, setResending] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  const inputRef = useRef<HTMLInputElement>(null);
  const verifyingRef = useRef(false);

  const resendIn = secondsUntil(sent.resend_available_at, now);
  useEffect(() => {
    if (resendIn <= 0) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [resendIn]);

  useEffect(() => inputRef.current?.focus(), []);

  async function verify(value: string) {
    // A ref, not state: auto-submit on the 6th digit and a click on the button can race.
    if (verifyingRef.current) return;
    if (value.length !== CODE_LENGTH) {
      setCodeError(`Enter the ${CODE_LENGTH}-digit code.`);
      return;
    }
    verifyingRef.current = true;
    setVerifying(true);
    setCodeError(undefined);
    setFormError(undefined);
    try {
      const session = await verifySignUp({ email, code: value });
      // An existing learner who used the code is simply signed in.
      router.replace(session.is_new_user ? AFTER_SIGN_UP : AFTER_SIGN_IN);
    } catch (err) {
      const view = describeAuthError(err);
      setCodeError(view.fields.code);
      setFormError(view.message);
      setCode("");
      inputRef.current?.focus();
      verifyingRef.current = false;
      setVerifying(false);
    }
  }

  async function resend() {
    if (resending || resendIn > 0) return;
    setResending(true);
    setFormError(undefined);
    setNotice(undefined);
    try {
      const next = await startSignUp({ email });
      setNow(Date.now());
      onResent(next);
      setNotice("We sent you a new code.");
    } catch (err) {
      setFormError(describeAuthError(err).message);
    } finally {
      setResending(false);
    }
  }

  return (
    <AuthCard
      icon={<MailIcon size={20} />}
      title="Check your email"
      description={
        <>
          We sent a {CODE_LENGTH}-digit code to <span className="font-medium text-fg">{email}</span>
          . It expires in 10 minutes.
        </>
      }
      footer={
        <button
          type="button"
          onClick={onChangeEmail}
          className="focus-ring inline-flex items-center gap-1.5 rounded-sm font-medium text-fg-muted hover:text-fg"
        >
          <ArrowLeftIcon size={14} />
          Use a different email
        </button>
      }
    >
      <form
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          void verify(code);
        }}
        className="flex flex-col gap-4"
        aria-label="Verify email"
      >
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        {notice && !formError ? <FormAlert variant="success">{notice}</FormAlert> : null}
        <FormField label="Verification code" error={codeError}>
          <Input
            ref={inputRef}
            name="code"
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9]*"
            maxLength={CODE_LENGTH + 4}
            placeholder="000000"
            value={code}
            disabled={verifying}
            onChange={(e) => {
              const value = sanitizeCode(e.target.value);
              setCode(value);
              if (value.length === CODE_LENGTH) void verify(value);
            }}
            className="h-(--control-lg) text-center text-xl font-semibold tracking-widest tabular-nums"
            data-testid="sign-up-code"
          />
        </FormField>
        <Button type="submit" size="lg" fullWidth loading={verifying} data-testid="sign-up-verify">
          Verify and continue
        </Button>
        <p className="text-center text-sm text-fg-muted" aria-live="polite">
          Didn&apos;t get it? Check spam, or{" "}
          {resendIn > 0 ? (
            <span>resend in {resendIn}s</span>
          ) : (
            <button
              type="button"
              onClick={() => void resend()}
              disabled={resending}
              className="focus-ring rounded-sm font-medium text-fg underline-offset-4 hover:underline disabled:opacity-50"
              data-testid="sign-up-resend"
            >
              {resending ? "sending…" : "resend the code"}
            </button>
          )}
          .
        </p>
      </form>
    </AuthCard>
  );
}
