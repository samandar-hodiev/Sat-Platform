"use client";

import { useState, type FormEvent } from "react";

import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  FormField,
  Input,
  Modal,
  useToast,
} from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import { describeAuthError } from "@/features/auth/errors";
import { refreshSession } from "@/features/auth/session";
import {
  CODE_LENGTH,
  normalizeEmail,
  sanitizeCode,
  validateEmail,
} from "@/features/auth/validation";

import { confirmEmailChange, startEmailChange, type SignInMethods } from "../api";
import { EMPTY_REAUTH, reauthBody, validateReauth, type ReauthValues } from "../reauth";
import { ReauthFields } from "./ReauthFields";

/** The account email, and changing it: confirm it's you → code to the new address → done. */
export function EmailCard({
  methods,
  onChanged,
}: {
  methods: SignInMethods;
  onChanged: (next: SignInMethods) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Card>
      <CardHeader>
        <CardTitle>Email</CardTitle>
        <CardDescription>You sign in and get study emails at this address.</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="truncate text-sm font-medium text-fg" data-testid="account-email">
            {methods.email}
          </span>
          {methods.email_verified ? (
            <Badge variant="success">Verified</Badge>
          ) : (
            <Badge variant="warning">Not verified</Badge>
          )}
        </div>
        <Button variant="outline" onClick={() => setOpen(true)}>
          Change email
        </Button>
      </CardContent>
      {open ? (
        <ChangeEmailModal
          methods={methods}
          onClose={() => setOpen(false)}
          onDone={(next) => {
            setOpen(false);
            onChanged(next);
          }}
        />
      ) : null}
    </Card>
  );
}

type Step = { kind: "request" } | { kind: "confirm"; newEmail: string };

function ChangeEmailModal({
  methods,
  onClose,
  onDone,
}: {
  methods: SignInMethods;
  onClose: () => void;
  onDone: (next: SignInMethods) => void;
}) {
  const { toast } = useToast();
  const [step, setStep] = useState<Step>({ kind: "request" });
  const [newEmail, setNewEmail] = useState("");
  const [reauth, setReauth] = useState<ReauthValues>(EMPTY_REAUTH);
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const [formError, setFormError] = useState<string>();
  const [busy, setBusy] = useState(false);

  function fail(err: unknown) {
    const view = describeAuthError(err);
    setErrors(view.fields);
    setFormError(view.message);
  }

  async function request(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const found = {
      new_email: validateEmail(newEmail),
      ...validateReauth(methods.reauth_method, reauth),
    };
    if (!found.new_email && normalizeEmail(newEmail) === methods.email)
      found.new_email = "This is already your email.";
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;
    setBusy(true);
    try {
      const started = await startEmailChange({
        new_email: normalizeEmail(newEmail),
        ...reauthBody(methods.reauth_method, reauth),
      });
      setStep({ kind: "confirm", newEmail: started.new_email });
      setErrors({});
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  async function confirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    if (code.length !== CODE_LENGTH) {
      setErrors({ code: `Enter the ${CODE_LENGTH}-digit code.` });
      return;
    }
    setBusy(true);
    setFormError(undefined);
    try {
      const next = await confirmEmailChange({ code });
      // The session keeps the user's email; pick up the new one (header, menus).
      await refreshSession().catch(() => false);
      toast({
        variant: "success",
        title: "Email changed",
        description: "Other devices were signed out. Sign in there with your new email.",
      });
      onDone(next);
    } catch (err) {
      fail(err);
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      dismissible={!busy}
      title="Change email"
      description={
        step.kind === "request"
          ? "We'll send a code to the new address. Your email changes once you enter it."
          : `Enter the code we sent to ${step.newEmail}.`
      }
    >
      {step.kind === "request" ? (
        <form
          noValidate
          onSubmit={request}
          className="flex flex-col gap-4"
          aria-label="Request an email change"
        >
          {formError ? <FormAlert>{formError}</FormAlert> : null}
          <FormField label="New email" required error={errors.new_email}>
            <Input
              type="email"
              name="email"
              autoComplete="email"
              value={newEmail}
              maxLength={254}
              disabled={busy}
              onChange={(e) => setNewEmail(e.target.value)}
            />
          </FormField>
          <ReauthFields
            method={methods.reauth_method}
            email={methods.email}
            values={reauth}
            onChange={setReauth}
            errors={errors}
            disabled={busy}
          />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" loading={busy} disabled={busy}>
              Send code
            </Button>
          </div>
        </form>
      ) : (
        <form
          noValidate
          onSubmit={confirm}
          className="flex flex-col gap-4"
          aria-label="Confirm new email"
        >
          {formError ? <FormAlert>{formError}</FormAlert> : null}
          <FormField label="Code" required hint="It expires in 10 minutes." error={errors.code}>
            <Input
              name="one-time-code"
              autoComplete="one-time-code"
              inputMode="numeric"
              placeholder="123456"
              value={code}
              maxLength={CODE_LENGTH}
              disabled={busy}
              onChange={(e) => setCode(sanitizeCode(e.target.value))}
            />
          </FormField>
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              disabled={busy}
              onClick={() => {
                setStep({ kind: "request" });
                setCode("");
                setReauth(EMPTY_REAUTH);
                setErrors({});
                setFormError(undefined);
              }}
            >
              Use another email
            </Button>
            <Button type="submit" loading={busy} disabled={busy}>
              Confirm email
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}
