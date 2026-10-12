"use client";

import { useRef, useState, type ChangeEvent } from "react";

import { Avatar, Button } from "@/components/ui";
import { ApiError } from "@/lib/api/client";

import { avatarSrc, removeAvatar, uploadAvatar, type LearnerProfile } from "../api";
import { AVATAR_TYPES, fieldMessage, validateAvatarFile } from "../validation";

/**
 * Profile picture with upload and remove. Without a photo the default avatar (initials, or "?"
 * before a name is typed) is shown. The file goes to the API right away; the server crops and
 * resizes it, so what is shown afterwards is exactly what others will see.
 */
export function AvatarPicker({
  name,
  avatarUrl,
  onChange,
}: {
  /** Used for the initials of the default avatar and the alt text. */
  name: string;
  avatarUrl: string | null;
  onChange: (profile: LearnerProfile) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState<"upload" | "remove" | null>(null);
  const [error, setError] = useState<string>();

  async function run(kind: "upload" | "remove", call: () => Promise<LearnerProfile>) {
    setBusy(kind);
    setError(undefined);
    try {
      onChange(await call());
    } catch (err) {
      setError(describe(err));
    } finally {
      setBusy(null);
    }
  }

  function onFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = ""; // choosing the same file again still triggers a change
    if (!file) return;
    const invalid = validateAvatarFile(file);
    if (invalid) {
      setError(invalid);
      return;
    }
    void run("upload", () => uploadAvatar(file));
  }

  const label = name.trim() || "?";
  return (
    <div className="flex items-center gap-4">
      <Avatar name={label} src={avatarSrc(avatarUrl)} size="xl" />
      <div className="flex min-w-0 flex-col gap-2">
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            loading={busy === "upload"}
            disabled={busy !== null}
            onClick={() => input.current?.click()}
          >
            {avatarUrl ? "Change photo" : "Upload photo"}
          </Button>
          {avatarUrl ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              loading={busy === "remove"}
              disabled={busy !== null}
              onClick={() => void run("remove", removeAvatar)}
            >
              Remove
            </Button>
          ) : null}
        </div>
        {error ? (
          <p role="alert" className="text-sm text-error-text">
            {error}
          </p>
        ) : (
          <p className="text-sm text-fg-muted">JPG, PNG or WebP, up to 5 MB. Optional.</p>
        )}
      </div>
      <input
        ref={input}
        type="file"
        accept={AVATAR_TYPES.join(",")}
        className="sr-only"
        tabIndex={-1}
        aria-label="Profile picture file"
        onChange={onFile}
        data-testid="avatar-input"
      />
    </div>
  );
}

function describe(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.fieldErrors.file) return fieldMessage(err.fieldErrors.file) ?? err.message;
    if (err.code === "PAYLOAD_TOO_LARGE") return "The image is larger than 5 MB.";
    if (err.code === "RATE_LIMITED") return "Too many uploads. Please try again later.";
    if (err.code === "NETWORK_ERROR") return err.message;
  }
  return "The photo could not be saved. Please try again.";
}
