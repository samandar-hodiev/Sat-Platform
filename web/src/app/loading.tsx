import { LoadingState } from "@/components/ui";

// Shared loading state: every route segment can override it with its own loading.tsx
// (preferably skeletons shaped like the final content).
export default function Loading() {
  return <LoadingState className="min-h-[50dvh]" />;
}
