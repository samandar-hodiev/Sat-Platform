"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { Button, FormField, Input, buttonClasses } from "@/components/ui";

import { requestPasswordReset } from "../api";
import { describeAuthError } from "../errors";
import { AUTH_ROUTES } from "../routes";
import { normalizeEmail, validateEmail } from "../validation";
import { AuthCard } from "./AuthCard";
import { FormAlert } from "./FormAlert";
import { ArrowLeftIcon, KeyIcon, MailIcon } from "./icons";

function BackToSignIn() {
  return (
    <Link
      href={AUTH_ROUTES.signIn}
      className="focus-ring inline-flex items-center gap-1.5 rounded-sm font-medium text-fg-muted hover:text-fg"
    >
      <ArrowLeftIcon size={14} />
      Back to sign in
    </Link>
  );
}

export function ForgotPasswordForm() {
  const [email, setEmail] = useState("");
  const [emailError, setEmailError] = useState<string>();
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);
  const [sentTo, setSentTo] = useState<string>();

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
      await requestPasswordReset({ email: normalized });
      setSentTo(normalized);
    } catch (err) {
      const view = describeAuthError(err);
      setEmailError(view.fields.email);
      setFormError(view.message);
    } finally {
      setSubmitting(false);
    }
  }

  if (sentTo) {
    return (
      <AuthCard
        icon={<MailIcon size={20} />}
        title="Check your email"
        description={
          <>
            If an account exists for <span className="font-medium text-fg">{sentTo}</span>, we sent
            a link to reset your password. The link expires in 1 hour.
          </>
        }
        footer={<BackToSignIn />}
      >
        <Link
          href={AUTH_ROUTES.signIn}
          className={buttonClasses({ variant: "outline", size: "lg", fullWidth: true })}
        >
          Return to sign in
        </Link>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      icon={<KeyIcon size={20} />}
      title="Forgot your password?"
      description="Enter your email and we'll send you a link to reset it."
      footer={<BackToSignIn />}
    >
      <form
        noValidate
        onSubmit={onSubmit}
        className="flex flex-col gap-4"
        aria-label="Reset password"
      >
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <FormField label="Email" error={emailError}>
          <Input
            type="email"
            name="email"
            autoComplete="email"
            inputMode="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            data-testid="forgot-email"
          />
        </FormField>
        <Button type="submit" size="lg" fullWidth loading={submitting} data-testid="forgot-submit">
          Send reset link
        </Button>
      </form>
    </AuthCard>
  );
}
