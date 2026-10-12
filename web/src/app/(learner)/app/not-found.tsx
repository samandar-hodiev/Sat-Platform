import Link from "next/link";

import { EmptyState, SearchIcon, buttonClasses } from "@/components/ui";
import { LEARNER_HOME } from "@/features/learner-shell";

export default function LearnerNotFound() {
  return (
    <EmptyState
      className="min-h-[50dvh] justify-center"
      icon={<SearchIcon size={20} />}
      title="Page not found"
      description="This page does not exist or is not available yet."
      action={
        <Link href={LEARNER_HOME} className={buttonClasses({ variant: "outline", size: "sm" })}>
          Go to the Dashboard
        </Link>
      }
    />
  );
}
