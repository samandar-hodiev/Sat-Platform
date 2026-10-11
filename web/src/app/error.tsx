"use client";

// Shared error state with a retry action. Never shows raw error messages: they can contain
// internal details. The request id (when present) helps support find the server log.
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div role="alert" className="flex flex-col items-start gap-3 p-6">
      <h2 className="text-lg font-semibold">Something went wrong</h2>
      <p>Please try again. If it keeps happening, contact support.</p>
      {error.digest ? <p className="text-sm">Reference: {error.digest}</p> : null}
      <button type="button" onClick={reset} className="rounded border px-3 py-1.5">
        Try again
      </button>
    </div>
  );
}
