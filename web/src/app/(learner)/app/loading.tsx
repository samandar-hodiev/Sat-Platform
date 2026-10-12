import { LoadingState } from "@/components/ui";

// Shown inside the shell while a learner page loads; pages may add skeleton loading.tsx files.
export default function LearnerLoading() {
  return <LoadingState className="min-h-[50dvh]" />;
}
