import Link from "next/link";

import { EmptyState, SearchIcon, buttonClasses } from "@/components/ui";

export default function ExamNotFound() {
  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <EmptyState
        icon={<SearchIcon size={20} />}
        title="Exam not found"
        description="This exam does not exist or is no longer available."
        action={
          <Link href="/app/exams" className={buttonClasses({ variant: "outline", size: "sm" })}>
            Back to Mock Exams
          </Link>
        }
      />
    </div>
  );
}
