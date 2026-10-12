import { notFound } from "next/navigation";

// Until 3.08 adds /exam/[attemptId], every /exam/* path is not found (inside the focus layout).
export default function MissingExamPage() {
  notFound();
}
