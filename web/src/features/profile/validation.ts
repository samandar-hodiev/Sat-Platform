// Client-side checks for instant feedback only; the API validates everything again.
// Rules match api/internal/profile (validate.go, avatar.go).

export const NAME_MAX_LENGTH = 100;
export const MINIMUM_AGE = 13;
export const AVATAR_MAX_BYTES = 5 * 1024 * 1024;
export const AVATAR_TYPES = ["image/jpeg", "image/png", "image/webp"] as const;
/** Prefilled in the phone field; left as is, it means "no phone". */
export const PHONE_PREFIX = "+998 ";

const NAME_PATTERN = /^\p{L}[\p{L}\p{M} '’.-]*$/u;

export function normalizeName(value: string): string {
  return value.trim().split(/\s+/).filter(Boolean).join(" ");
}

export function validateName(value: string, label: "first" | "last"): string | undefined {
  const name = normalizeName(value);
  if (!name) return `Enter your ${label} name.`;
  if ([...name].length > NAME_MAX_LENGTH) return `Use at most ${NAME_MAX_LENGTH} characters.`;
  if (!NAME_PATTERN.test(name)) return "Use letters, spaces, hyphens, apostrophes and dots only.";
  return undefined;
}

/** The phone as the API expects it: "" when empty or only the default prefix. */
export function phoneForApi(value: string): string {
  const v = value.trim();
  return v === "" || v === PHONE_PREFIX.trim() ? "" : v;
}

export function validatePhone(value: string): string | undefined {
  const v = phoneForApi(value);
  if (!v) return undefined;
  if (!/^[+0-9 ()\-.]+$/.test(v)) return "Use digits only, e.g. +998 90 123 45 67.";
  let digits = v.replace(/[^\d+]/g, "");
  if (digits.startsWith("00")) digits = `+${digits.slice(2)}`;
  if (!/^\+[1-9]\d{6,14}$/.test(digits)) {
    return "Enter an international number, e.g. +998 90 123 45 67.";
  }
  return undefined;
}

function isoDate(d: Date): string {
  const y = String(d.getFullYear()).padStart(4, "0");
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

/** Latest date of birth allowed today (the learner turns MINIMUM_AGE today). */
export function latestBirthDate(today: Date = new Date()): string {
  const d = new Date(today.getFullYear() - MINIMUM_AGE, today.getMonth(), today.getDate());
  // 29 Feb today → 28 Feb or 1 Mar in a non-leap year: Date rolls over to 1 Mar, which is later
  // than allowed; step back to the last day of February.
  if (d.getMonth() !== today.getMonth()) d.setDate(0);
  return isoDate(d);
}

export function validateDateOfBirth(value: string, today: Date = new Date()): string | undefined {
  if (!value) return "Enter your date of birth.";
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  const date = match ? new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3])) : null;
  if (!match || !date || isoDate(date) !== value) return "Enter a valid date.";
  if (Number(match[1]) < 1900 || value > isoDate(today)) return "Enter your real date of birth.";
  if (value > latestBirthDate(today)) {
    return `You must be at least ${MINIMUM_AGE} years old to use SAT Platform.`;
  }
  return undefined;
}

export function validateAvatarFile(file: File): string | undefined {
  if (!(AVATAR_TYPES as readonly string[]).includes(file.type)) {
    return "Upload a JPG, PNG or WebP image.";
  }
  if (file.size > AVATAR_MAX_BYTES) return "The image is larger than 5 MB.";
  return undefined;
}

/** The browser's IANA time zone, "" if unknown (the API then keeps its default). */
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone ?? "";
  } catch {
    return "";
  }
}

/** Server field messages are fragments ("is required"); make them sentences for the form. */
export function fieldMessage(message: string | undefined): string | undefined {
  if (!message) return undefined;
  const m = message.charAt(0).toUpperCase() + message.slice(1);
  return /[.!?]$/.test(m) ? m : `${m}.`;
}
