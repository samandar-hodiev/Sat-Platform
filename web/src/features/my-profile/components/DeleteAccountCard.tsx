"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import {
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
import { signOut } from "@/features/auth/session";
import { fieldMessage } from "@/features/profile/validation";

import { deleteAccount, type SignInMethods } from "../api";
import { EMPTY_REAUTH, reauthBody, validateReauth, type ReauthValues } from "../reauth";
import { AFTER_ACCOUNT_DELETED, DELETE_CONFIRMATION } from "../routes";
import { ReauthFields } from "./ReauthFields";

/** Danger zone: deleting the account at once and for good (decided for 2.10). */
export function DeleteAccountCard({ methods }: { methods: SignInMethods }) {
  const [open, setOpen] = useState(false);
  return (
    <Card className="border-error/40">
      <CardHeader>
        <CardTitle className="text-error-text">Delete account</CardTitle>
        <CardDescription>
          Permanently delete your account and personal details. This can&apos;t be undone.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex justify-end">
        <Button variant="danger" onClick={() => setOpen(true)}>
          Delete account
        </Button>
      </CardContent>
      {open ? <DeleteAccountModal methods={methods} onClose={() => setOpen(false)} /> : null}
    </Card>
  );
}

function DeleteAccountModal({ methods, onClose }: { methods: SignInMethods; onClose: () => void }) {
  const router = useRouter();
  const { toast } = useToast();
  const [confirmation, setConfirmation] = useState("");
  const [reauth, setReauth] = useState<ReauthValues>(EMPTY_REAUTH);
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const [formError, setFormError] = useState<string>();
  const [busy, setBusy] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const found: Record<string, string | undefined> = {
      confirmation:
        confirmation === DELETE_CONFIRMATION
          ? undefined
          : `Type ${DELETE_CONFIRMATION} to confirm.`,
      ...validateReauth(methods.reauth_method, reauth),
    };
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;
    setBusy(true);
    try {
      await deleteAccount({ confirmation, ...reauthBody(methods.reauth_method, reauth) });
    } catch (err) {
      const view = describeAuthError(err);
      setErrors({ ...view.fields, confirmation: fieldMessage(view.fields.confirmation) });
      setFormError(view.message);
      setBusy(false);
      return;
    }
    // Every session is already revoked on the server; clear this one (cookie included).
    await signOut().catch(() => undefined);
    toast({ variant: "success", title: "Your account was deleted" });
    router.replace(AFTER_ACCOUNT_DELETED);
  }

  return (
    <Modal open onClose={onClose} dismissible={!busy} title="Delete your account?">
      <form
        noValidate
        onSubmit={onSubmit}
        className="flex flex-col gap-4"
        aria-label="Delete account"
      >
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <div className="flex flex-col gap-2 text-sm text-fg-muted">
          <p>This happens right away and can&apos;t be undone:</p>
          <ul className="list-disc pl-5">
            <li>You are signed out on every device.</li>
            <li>Your name, phone, date of birth and photo are deleted.</li>
            <li>Your password and linked Google account are removed.</li>
            <li>{methods.email} is freed and can be used for a new account.</li>
          </ul>
        </div>
        <FormField
          label={`Type ${DELETE_CONFIRMATION} to confirm`}
          required
          error={errors.confirmation}
        >
          <Input
            name="confirmation"
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            value={confirmation}
            maxLength={32}
            disabled={busy}
            onChange={(e) => setConfirmation(e.target.value)}
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
          <Button type="submit" variant="danger" loading={busy} disabled={busy}>
            Delete my account
          </Button>
        </div>
      </form>
    </Modal>
  );
}
