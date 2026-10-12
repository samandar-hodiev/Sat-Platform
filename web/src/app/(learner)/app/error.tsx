"use client";

import { ErrorState } from "@/components/ui";

// Keeps the sidebar and header when a learner page fails, so the learner can navigate away.
export default function LearnerError({
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
