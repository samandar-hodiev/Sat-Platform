import Link from "next/link";

import { EmptyState, SearchIcon, buttonClasses } from "@/components/ui";

export default function NotFound() {
  return (
    <main className="flex min-h-dvh items-center justify-center p-6">
      <EmptyState
        icon={<SearchIcon size={20} />}
        title="Page not found"
        description="The page you are looking for does not exist or has moved."
        action={
          <Link href="/" className={buttonClasses({ variant: "outline", size: "sm" })}>
            Go to the home page
          </Link>
        }
      />
    </main>
  );
}
