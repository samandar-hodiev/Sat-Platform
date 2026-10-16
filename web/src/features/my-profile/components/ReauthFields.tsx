"use client";

import { useEffect, useState } from "react";

import { Button, FormField, Input } from "@/components/ui";
import { describeAuthError } from "@/features/auth/errors";
import { PasswordInput } from "@/features/auth/components/PasswordInput";
import { CODE_LENGTH, sanitizeCode } from "@/features/auth/validation";

import { sendReauthCode } from "../api";
import { secondsUntil, type ReauthMethod, type ReauthValues } from "../reauth";

/**
 * "Confirm it's you" part of a sensitive form: the current password, or (accounts without one)
 * a button that emails a code and the field to type it in.
 */
export function ReauthFields({
  method,
  email,
  values,
  onChange,
  errors,
  disabled,
}: {
  method: ReauthMethod;
  /** Where the code goes (the current account email). */
  email: string;
  values: ReauthValues;
  onChange: (next: ReauthValues) => void;
  errors: Partial<Record<"current_password" | "code", string>>;
  disabled?: boolean;
}) {
  if (method === "password") {
    return (
      <FormField label="Current password" required error={errors.current_password}>
        <PasswordInput
          name="current-password"
          autoComplete="current-password"
          value={values.password}
          maxLength={512}
          disabled={disabled}
          onChange={(e) => onChange({ ...values, password: e.target.value })}
        />
      </FormField>
    );
  }
  return (
    <CodeField
      email={email}
      value={values.code}
      onChange={(code) => onChange({ ...values, code })}
      error={errors.code}
      disabled={disabled}
    />
  );
}

function CodeField({
  email,
  value,
  onChange,
  error,
  disabled,
}: {
  email: string;
  value: string;
  onChange: (code: string) => void;
  error?: string;
  disabled?: boolean;
}) {
  const [sending, setSending] = useState(false);
  const [resendAt, setResendAt] = useState<string>();
  const [sendError, setSendError] = useState<string>();
  const [wait, setWait] = useState(0);

  useEffect(() => {
    if (!resendAt) return;
    const tick = () => setWait(secondsUntil(resendAt));
    tick();
    const timer = window.setInterval(tick, 1000);
    return () => window.clearInterval(timer);
  }, [resendAt]);

  async function send() {
    setSending(true);
    setSendError(undefined);
    try {
      const sent = await sendReauthCode();
      setResendAt(sent.resend_available_at);
    } catch (err) {
      setSendError(describeAuthError(err).message ?? "The code could not be sent.");
    } finally {
      setSending(false);
    }
  }

  const sent = resendAt !== undefined;
  const hint = sent
    ? `We sent a ${CODE_LENGTH}-digit code to ${email}. It expires in 10 minutes.`
    : `We'll email a ${CODE_LENGTH}-digit code to ${email} to confirm it's you.`;

  return (
    <div className="flex flex-col gap-2">
      <FormField label="Confirmation code" required hint={hint} error={sendError ?? error}>
        <Input
          name="one-time-code"
          autoComplete="one-time-code"
          inputMode="numeric"
          placeholder="123456"
          value={value}
          maxLength={CODE_LENGTH}
          disabled={disabled || !sent}
          onChange={(e) => onChange(sanitizeCode(e.target.value))}
        />
      </FormField>
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          loading={sending}
          disabled={disabled || sending || wait > 0}
          onClick={send}
        >
          {!sent ? "Email me a code" : wait > 0 ? `Resend code in ${wait}s` : "Resend code"}
        </Button>
      </div>
    </div>
  );
}
