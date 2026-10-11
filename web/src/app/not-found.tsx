import Link from "next/link";

export default function NotFound() {
  return (
    <main className="flex flex-col items-start gap-3 p-6">
      <h1 className="text-lg font-semibold">Page not found</h1>
      <Link href="/" className="underline">
        Go to the home page
      </Link>
    </main>
  );
}
