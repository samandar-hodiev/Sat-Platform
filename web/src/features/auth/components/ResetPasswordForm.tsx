"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import {
  AlertCircleIcon,
  Button,
  CheckCircleIcon,
  FormField,
  buttonClasses,
} from "@/components/ui";

import { resetPassword } from "../api";
import { describeAuthError } from "../errors";
import { AUTH_ROUTES } from "../routes";
import { PASSWORD_MIN_LENGTH, validateNewPassword } from "../validation";
import { AuthCard } from "./AuthCard";
import { FormAlert } from "./FormAlert";
import { KeyIcon } from "./icons";
import { PasswordInput } from "./PasswordInput";

type Fields = Partial<Record<"password" | "confirm", string>>;

/** `token` comes from the emailed link (?token=…). Without one the link is shown as invalid. */
export function ResetPasswordForm({ token }: { token: string | undefined }) {
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Fields>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  if (!token) {
    return (
      <AuthCard
        icon={<AlertCircleIcon size={20} />}
        title="This link is not valid"
        description="The password reset link is incomplete or has expired. Request a new one."
      >
        <Link
          href={AUTH_ROUTES.forgotPassword}
          className={buttonClasses({ size: "lg", fullWidth: true })}
        >
          Request a new link
        </Link>
      </AuthCard>
    );
  }

  if (done) {
    return (
      <AuthCard
        icon={<CheckCircleIcon size={20} />}
        title="Password updated"
        description="Your password has been changed and you were signed out on all devices. Sign in with your new password."
      >
        <Link href={AUTH_ROUTES.signIn} className={buttonClasses({ size: "lg", fullWidth: true })}>
          Sign in
        </Link>
      </AuthCard>
    );
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting || !token) return;
    const errors: Fields = {
      password: validateNewPassword(password),
      confirm: confirm === password ? undefined : "Passwords do not match.",
    };
    setFieldErrors(errors);
    setFormError(undefined);
    if (errors.password || errors.confirm) return;

    setSubmitting(true);
    try {
      await resetPassword({ token, password });
      setDone(true);
    } catch (err) {
      const view = describeAuthError(err);
      setFieldErrors({ password: view.fields.password });
      setFormError(view.message ?? (view.fields.token ? view.fields.token : undefined));
      setSubmitting(false);
    }
  }

  return (
    <AuthCard
      icon={<KeyIcon size={20} />}
      title="Set a new password"
      description="Choose a password you don't use on other sites."
    >
      <form
        noValidate
        onSubmit={onSubmit}
        className="flex flex-col gap-4"
        aria-label="Set a new password"
      >
        {formError ? (
          <FormAlert>
            {formError}{" "}
            <Link
              href={AUTH_ROUTES.forgotPassword}
              className="font-medium underline underline-offset-4"
            >
              Request a new link
            </Link>
          </FormAlert>
        ) : null}
        <FormField
          label="New password"
          hint={`At least ${PASSWORD_MIN_LENGTH} characters.`}
          error={fieldErrors.password}
        >
          <PasswordInput
            name="new-password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            data-testid="reset-password"
          />
        </FormField>
        <FormField label="Confirm new password" error={fieldErrors.confirm}>
          <PasswordInput
            name="confirm-password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            data-testid="reset-confirm"
          />
        </FormField>
        <Button type="submit" size="lg" fullWidth loading={submitting} data-testid="reset-submit">
          Update password
        </Button>
      </form>
    </AuthCard>
  );
}
