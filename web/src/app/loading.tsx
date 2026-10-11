// Shared loading state: every route segment can override it with its own loading.tsx.
export default function Loading() {
  return (
    <div role="status" aria-live="polite" className="p-6">
      Loading…
    </div>
  );
}
