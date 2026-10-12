"use client";

import { ErrorState } from "@/components/ui";

// Shared error state with a retry action. Never shows raw error messages: they can contain
// internal details. The digest (when present) helps support find the server log.
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <ErrorState className="min-h-[50dvh] justify-center" onRetry={reset} reference={error.digest} />
  );
}
