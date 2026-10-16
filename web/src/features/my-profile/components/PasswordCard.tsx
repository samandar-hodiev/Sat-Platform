"use client";

import { useState, type FormEvent } from "react";

import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  FormField,
  Modal,
  useToast,
} from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import { PasswordInput } from "@/features/auth/components/PasswordInput";
import { describeAuthError } from "@/features/auth/errors";
import { refreshSession } from "@/features/auth/session";
import {
  PASSWORD_MAX_LENGTH,
  PASSWORD_MIN_LENGTH,
  validateNewPassword,
} from "@/features/auth/validation";

import { changePassword, type SignInMethods } from "../api";
import { EMPTY_REAUTH, reauthBody, validateReauth, type ReauthValues } from "../reauth";
import { ReauthFields } from "./ReauthFields";

/** Change the password, or set a first one (Google and email-code accounts). */
export function PasswordCard({
  methods,
  onChanged,
}: {
  methods: SignInMethods;
  onChanged: () => void;
}) {
  const [open, setOpen] = useState(false);
  const has = methods.has_password;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Password</CardTitle>
        <CardDescription>
          {has
            ? "Changing your password signs you out on your other devices."
            : "You don't have a password yet. Add one to sign in with your email and password."}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex justify-end">
        <Button variant="outline" onClick={() => setOpen(true)}>
          {has ? "Change password" : "Set a password"}
        </Button>
      </CardContent>
      {open ? (
        <PasswordModal
          methods={methods}
          onClose={() => setOpen(false)}
          onDone={() => {
            setOpen(false);
            onChanged();
          }}
        />
      ) : null}
    </Card>
  );
}

function PasswordModal({
  methods,
  onClose,
  onDone,
}: {
  methods: SignInMethods;
  onClose: () => void;
  onDone: () => void;
}) {
  const { toast } = useToast();
  const [reauth, setReauth] = useState<ReauthValues>(EMPTY_REAUTH);
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const [formError, setFormError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const has = methods.has_password;

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const found: Record<string, string | undefined> = {
      ...validateReauth(methods.reauth_method, reauth),
      new_password: validateNewPassword(password),
      confirm: confirm === password ? undefined : "Passwords do not match.",
    };
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;
    setBusy(true);
    try {
      await changePassword({
        new_password: password,
        ...reauthBody(methods.reauth_method, reauth),
      });
      await refreshSession().catch(() => false);
      toast({
        variant: "success",
        title: has ? "Password changed" : "Password set",
        description: "Other devices were signed out.",
      });
      onDone();
    } catch (err) {
      const view = describeAuthError(err);
      setErrors(view.fields);
      setFormError(view.message);
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      dismissible={!busy}
      title={has ? "Change password" : "Set a password"}
    >
      <form noValidate onSubmit={onSubmit} className="flex flex-col gap-4" aria-label="Password">
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <ReauthFields
          method={methods.reauth_method}
          email={methods.email}
          values={reauth}
          onChange={setReauth}
          errors={errors}
          disabled={busy}
        />
        <FormField
          label="New password"
          required
          hint={`At least ${PASSWORD_MIN_LENGTH} characters.`}
          error={errors.new_password}
        >
          <PasswordInput
            name="new-password"
            autoComplete="new-password"
            value={password}
            maxLength={PASSWORD_MAX_LENGTH}
            disabled={busy}
            onChange={(e) => setPassword(e.target.value)}
          />
        </FormField>
        <FormField label="Confirm new password" required error={errors.confirm}>
          <PasswordInput
            name="confirm-password"
            autoComplete="new-password"
            value={confirm}
            maxLength={PASSWORD_MAX_LENGTH}
            disabled={busy}
            onChange={(e) => setConfirm(e.target.value)}
          />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={busy} disabled={busy}>
            {has ? "Change password" : "Set password"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
