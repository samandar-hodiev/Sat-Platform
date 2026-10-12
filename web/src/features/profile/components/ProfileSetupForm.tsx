"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Button, Card, FormField, Input } from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import { PasswordInput } from "@/features/auth/components/PasswordInput";
import { PASSWORD_MIN_LENGTH, validateNewPassword } from "@/features/auth/validation";
import { ApiError } from "@/lib/api/client";

import { completeProfileSetup, type LearnerProfile } from "../api";
import { setProfile } from "../profileStore";
import { AFTER_PROFILE_SETUP } from "../routes";
import {
  PHONE_PREFIX,
  browserTimezone,
  fieldMessage,
  latestBirthDate,
  normalizeName,
  phoneForApi,
  validateDateOfBirth,
  validateName,
  validatePhone,
} from "../validation";
import { AvatarPicker } from "./AvatarPicker";

type Field =
  "first_name" | "last_name" | "phone" | "date_of_birth" | "timezone" | "password" | "confirm";
type Errors = Partial<Record<Field, string>>;

/** Profile Setup (PRD §7). `profile` is the learner's current, not yet completed profile. */
export function ProfileSetupForm({
  userId,
  profile: initial,
}: {
  userId: string;
  profile: LearnerProfile;
}) {
  const router = useRouter();
  const [profile, setLocal] = useState(initial);
  const [firstName, setFirstName] = useState(initial.first_name ?? "");
  const [lastName, setLastName] = useState(initial.last_name ?? "");
  const [phone, setPhone] = useState(initial.phone ?? PHONE_PREFIX);
  const [dob, setDob] = useState(initial.date_of_birth ?? "");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<Errors>({});
  const [formError, setFormError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  const askPassword = !profile.has_password;
  const passwordRequired = profile.password_required;

  function onAvatar(next: LearnerProfile) {
    setLocal(next);
    setProfile(userId, next);
  }

  function validate(): Errors {
    const e: Errors = {
      first_name: validateName(firstName, "first"),
      last_name: validateName(lastName, "last"),
      phone: validatePhone(phone),
      date_of_birth: validateDateOfBirth(dob),
    };
    if (askPassword && (passwordRequired || password || confirm)) {
      e.password = validateNewPassword(password);
      e.confirm = confirm === password ? undefined : "Passwords do not match.";
    }
    return e;
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    const found = validate();
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;

    setSubmitting(true);
    try {
      const saved = await completeProfileSetup({
        first_name: normalizeName(firstName),
        last_name: normalizeName(lastName),
        phone: phoneForApi(phone),
        date_of_birth: dob,
        timezone: browserTimezone(),
        password: askPassword ? password : "",
      });
      setProfile(userId, saved);
      router.replace(AFTER_PROFILE_SETUP);
    } catch (err) {
      setSubmitting(false);
      if (err instanceof ApiError && err.code === "CONFLICT") {
        // Finished in another tab, or the password was set meanwhile: go on with fresh data.
        if (err.details?.reason === "profile_already_completed") {
          router.replace(AFTER_PROFILE_SETUP);
          return;
        }
        setFormError(err.message);
        return;
      }
      if (err instanceof ApiError && Object.keys(err.fieldErrors).length > 0) {
        const f = err.fieldErrors;
        setErrors({
          first_name: fieldMessage(f.first_name),
          last_name: fieldMessage(f.last_name),
          phone: fieldMessage(f.phone),
          date_of_birth: fieldMessage(f.date_of_birth),
          password: fieldMessage(f.password),
        });
        if (f.timezone) setFormError("Your device time zone was not recognised. Please try again.");
        return;
      }
      setFormError(
        err instanceof ApiError && err.code === "NETWORK_ERROR"
          ? err.message
          : "Your profile could not be saved. Please try again.",
      );
    }
  }

  const fullName = [normalizeName(firstName), normalizeName(lastName)].filter(Boolean).join(" ");

  return (
    <Card className="flex flex-col gap-6 p-6 shadow-sm sm:p-8">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight text-fg">Set up your profile</h1>
        <p className="text-sm text-fg-muted">
          Tell us a little about yourself. Fields marked * are required.
        </p>
      </div>

      <AvatarPicker name={fullName} avatarUrl={profile.avatar_url} onChange={onAvatar} />

      <form noValidate onSubmit={onSubmit} className="flex flex-col gap-4" aria-label="Profile">
        {formError ? <FormAlert>{formError}</FormAlert> : null}
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField label="First name" required error={errors.first_name}>
            <Input
              name="given-name"
              autoComplete="given-name"
              value={firstName}
              maxLength={200}
              onChange={(e) => setFirstName(e.target.value)}
            />
          </FormField>
          <FormField label="Last name" required error={errors.last_name}>
            <Input
              name="family-name"
              autoComplete="family-name"
              value={lastName}
              maxLength={200}
              onChange={(e) => setLastName(e.target.value)}
            />
          </FormField>
        </div>
        <FormField label="Email" hint="Verified. You can change it later in My Profile.">
          <Input type="email" value={profile.email} readOnly aria-readonly="true" />
        </FormField>
        <FormField
          label="Phone number"
          hint="Optional. International format, e.g. +998 90 123 45 67."
          error={errors.phone}
        >
          <Input
            type="tel"
            name="tel"
            autoComplete="tel"
            inputMode="tel"
            value={phone}
            maxLength={32}
            onChange={(e) => setPhone(e.target.value)}
          />
        </FormField>
        <FormField
          label="Date of birth"
          required
          hint="You must be at least 13 years old."
          error={errors.date_of_birth}
        >
          <Input
            type="date"
            name="bday"
            autoComplete="bday"
            min="1900-01-01"
            max={latestBirthDate()}
            value={dob}
            onChange={(e) => setDob(e.target.value)}
          />
        </FormField>

        {askPassword ? (
          <fieldset className="flex flex-col gap-4">
            <legend className="mb-1 text-sm text-fg-muted">
              {passwordRequired
                ? "Create a password to sign in with your email."
                : "Optional: add a password to also sign in with your email. You can keep using Google."}
            </legend>
            <FormField
              label="Password"
              required={passwordRequired}
              hint={`At least ${PASSWORD_MIN_LENGTH} characters.`}
              error={errors.password}
            >
              <PasswordInput
                name="new-password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </FormField>
            <FormField label="Confirm password" required={passwordRequired} error={errors.confirm}>
              <PasswordInput
                name="confirm-password"
                autoComplete="new-password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </FormField>
          </fieldset>
        ) : null}

        <Button type="submit" size="lg" fullWidth loading={submitting}>
          Continue
        </Button>
      </form>
    </Card>
  );
}
