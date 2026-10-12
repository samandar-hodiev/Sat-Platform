"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Button, Checkbox, FormField, Input } from "@/components/ui";

import { signIn } from "../api";
import { describeAuthError } from "../errors";
import { AFTER_SIGN_IN, AUTH_ROUTES } from "../routes";
import { normalizeEmail, safeNextPath, validateEmail } from "../validation";
import { AuthCard } from "./AuthCard";
import { FormAlert } from "./FormAlert";
import { GoogleButton, OrDivider } from "./GoogleButton";
import { LegalNote } from "./LegalNote";
import { PasswordInput } from "./PasswordInput";

type Fields = Partial<Record<"email" | "password", string>>;

export function SignInForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(true);
  const [fieldErrors, setFieldErrors] = useState<Fields>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;

    const errors: Fields = {
      email: validateEmail(email),
      password: password ? undefined : "Enter your password.",
    };
    setFieldErrors(errors);
    setFormError(undefined);
    if (errors.email || errors.password) return;

    setSubmitting(true);
    try {
      await signIn({ email: normalizeEmail(email), password, remember });
      router.replace(safeNextPath(searchParams.get("next"), AFTER_SIGN_IN));
    } catch (error) {
      const view = describeAuthError(error);
      setFieldErrors(view.fields);
      setFormError(view.message);
      setSubmitting(false);
    }
  }

  return (
    <AuthCard
      title="Welcome back"
      description="Sign in to continue your SAT preparation."
      footer={
        <>
          <p>
            Don&apos;t have an account?{" "}
            <Link
              href={AUTH_ROUTES.signUp}
              className="font-medium text-fg underline-offset-4 hover:underline"
            >
              Create account
            </Link>
          </p>
          <LegalNote />
        </>
      }
    >
      <GoogleButton next={safeNextPath(searchParams.get("next"), AFTER_SIGN_IN)} />
      <OrDivider />
      <form noValidate onSubmit={onSubmit} className="flex flex-col gap-4" aria-label="Sign in">
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <FormField label="Email" error={fieldErrors.email}>
          <Input
            type="email"
            name="email"
            autoComplete="email"
            inputMode="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            data-testid="sign-in-email"
          />
        </FormField>
        <FormField label="Password" error={fieldErrors.password}>
          <PasswordInput
            name="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            data-testid="sign-in-password"
          />
        </FormField>
        <div className="flex items-center justify-between gap-3">
          <Checkbox
            label="Remember me"
            checked={remember}
            onChange={(e) => setRemember(e.target.checked)}
          />
          <Link
            href={AUTH_ROUTES.forgotPassword}
            className="focus-ring rounded-sm text-sm font-medium text-fg-muted hover:text-fg"
          >
            Forgot password?
          </Link>
        </div>
        <Button type="submit" size="lg" fullWidth loading={submitting} data-testid="sign-in-submit">
          Sign in
        </Button>
      </form>
    </AuthCard>
  );
}
