"use client";

import { useMemo, useState, type FormEvent } from "react";

import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  FormField,
  Input,
  Select,
  useToast,
} from "@/components/ui";
import { FormAlert } from "@/features/auth/components/FormAlert";
import type { LearnerProfile } from "@/features/profile/api";
import { AvatarPicker } from "@/features/profile/components/AvatarPicker";
import { setProfile } from "@/features/profile/profileStore";
import {
  PHONE_PREFIX,
  fieldMessage,
  latestBirthDate,
  normalizeName,
  phoneForApi,
  validateDateOfBirth,
  validateName,
  validatePhone,
} from "@/features/profile/validation";
import { ApiError } from "@/lib/api/client";

import { updateProfile, type ProfileUpdate } from "../api";

type Field = "first_name" | "last_name" | "phone" | "date_of_birth" | "timezone";
type Errors = Partial<Record<Field, string>>;

/** IANA zones the browser knows, with the current one first if it is missing. */
function timezoneOptions(current: string): string[] {
  let zones: string[] = [];
  try {
    zones = Intl.supportedValuesOf("timeZone");
  } catch {
    zones = [];
  }
  return zones.includes(current) ? zones : [current, ...zones];
}

/** Phone as stored (E.164) → as shown: the field shows the default prefix when there is none. */
function phoneField(phone: string | null): string {
  return phone ?? PHONE_PREFIX;
}

/** Personal details and avatar (PRD §20 Account). Saves only the fields that changed. */
export function ProfileDetailsCard({
  userId,
  profile,
}: {
  userId: string;
  profile: LearnerProfile;
}) {
  const { toast } = useToast();
  const [firstName, setFirstName] = useState(profile.first_name ?? "");
  const [lastName, setLastName] = useState(profile.last_name ?? "");
  const [phone, setPhone] = useState(phoneField(profile.phone));
  const [dob, setDob] = useState(profile.date_of_birth ?? "");
  const [timezone, setTimezone] = useState(profile.timezone);
  const [errors, setErrors] = useState<Errors>({});
  const [formError, setFormError] = useState<string>();
  const [saving, setSaving] = useState(false);
  const zones = useMemo(() => timezoneOptions(profile.timezone), [profile.timezone]);

  const changes: ProfileUpdate = {};
  if (normalizeName(firstName) !== (profile.first_name ?? ""))
    changes.first_name = normalizeName(firstName);
  if (normalizeName(lastName) !== (profile.last_name ?? ""))
    changes.last_name = normalizeName(lastName);
  if (phoneForApi(phone) !== phoneForApi(phoneField(profile.phone)))
    changes.phone = phoneForApi(phone);
  if (dob !== (profile.date_of_birth ?? "")) changes.date_of_birth = dob;
  if (timezone !== profile.timezone) changes.timezone = timezone;
  const dirty = Object.keys(changes).length > 0;

  function reset(p: LearnerProfile) {
    setFirstName(p.first_name ?? "");
    setLastName(p.last_name ?? "");
    setPhone(phoneField(p.phone));
    setDob(p.date_of_birth ?? "");
    setTimezone(p.timezone);
    setErrors({});
    setFormError(undefined);
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (saving || !dirty) return;
    const found: Errors = {
      first_name: validateName(firstName, "first"),
      last_name: validateName(lastName, "last"),
      phone: validatePhone(phone),
      date_of_birth: validateDateOfBirth(dob),
    };
    setErrors(found);
    setFormError(undefined);
    if (Object.values(found).some(Boolean)) return;

    setSaving(true);
    try {
      const saved = await updateProfile(changes);
      setProfile(userId, saved);
      reset(saved);
      toast({ variant: "success", title: "Profile saved" });
    } catch (err) {
      if (err instanceof ApiError && Object.keys(err.fieldErrors).length > 0) {
        const f = err.fieldErrors;
        setErrors({
          first_name: fieldMessage(f.first_name),
          last_name: fieldMessage(f.last_name),
          phone: fieldMessage(f.phone),
          date_of_birth: fieldMessage(f.date_of_birth),
          timezone: fieldMessage(f.timezone),
        });
      } else {
        setFormError(
          err instanceof ApiError && (err.code === "NETWORK_ERROR" || err.code === "CONFLICT")
            ? err.message
            : "Your changes could not be saved. Please try again.",
        );
      }
    } finally {
      setSaving(false);
    }
  }

  const fullName = [profile.first_name, profile.last_name].filter(Boolean).join(" ");

  return (
    <Card>
      <CardHeader>
        <CardTitle>Personal details</CardTitle>
        <CardDescription>
          Your name and photo are shown in your account. Fields marked * are required.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <AvatarPicker
          name={fullName}
          avatarUrl={profile.avatar_url}
          onChange={(p) => setProfile(userId, p)}
        />

        <form
          noValidate
          onSubmit={onSubmit}
          className="flex flex-col gap-4"
          aria-label="Personal details"
        >
          {formError ? <FormAlert>{formError}</FormAlert> : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label="First name" required error={errors.first_name}>
              <Input
                name="given-name"
                autoComplete="given-name"
                value={firstName}
                maxLength={200}
                disabled={saving}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </FormField>
            <FormField label="Last name" required error={errors.last_name}>
              <Input
                name="family-name"
                autoComplete="family-name"
                value={lastName}
                maxLength={200}
                disabled={saving}
                onChange={(e) => setLastName(e.target.value)}
              />
            </FormField>
            <FormField
              label="Phone number"
              hint="Optional. Clear the number to remove it."
              error={errors.phone}
            >
              <Input
                name="tel"
                type="tel"
                autoComplete="tel"
                inputMode="tel"
                value={phone}
                maxLength={32}
                disabled={saving}
                onChange={(e) => setPhone(e.target.value)}
              />
            </FormField>
            <FormField label="Date of birth" required error={errors.date_of_birth}>
              <Input
                name="bday"
                type="date"
                autoComplete="bday"
                min="1900-01-01"
                max={latestBirthDate()}
                value={dob}
                disabled={saving}
                onChange={(e) => setDob(e.target.value)}
              />
            </FormField>
            <FormField
              label="Time zone"
              hint="Used for your daily study plan and reminders."
              error={errors.timezone}
              className="sm:col-span-2"
            >
              <Select
                value={timezone}
                disabled={saving}
                onChange={(e) => setTimezone(e.target.value)}
              >
                {zones.map((z) => (
                  <option key={z} value={z}>
                    {z.replaceAll("_", " ")}
                  </option>
                ))}
              </Select>
            </FormField>
          </div>
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              disabled={!dirty || saving}
              onClick={() => reset(profile)}
            >
              Discard changes
            </Button>
            <Button type="submit" loading={saving} disabled={!dirty || saving}>
              Save changes
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
