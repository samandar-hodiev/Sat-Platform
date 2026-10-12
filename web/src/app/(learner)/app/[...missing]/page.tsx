import { notFound } from "next/navigation";

// Unknown /app/* paths (including pages not built yet) show the learner not-found page inside
// the shell instead of the bare root 404. Real routes always win over this catch-all.
export default function MissingLearnerPage() {
  notFound();
}
