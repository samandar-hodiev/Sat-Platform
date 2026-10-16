import { PublicFooter } from "@/features/landing/components/PublicFooter";
import { PublicHeader } from "@/features/landing/components/PublicHeader";

// Public marketing site (session 0.03): landing, pricing, FAQ, legal pages and contact.
export default function PublicLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="flex min-h-dvh flex-col bg-canvas text-fg">
      <a
        href="#main"
        className="focus-ring sr-only z-(--z-tooltip) rounded-md bg-surface px-3 py-2 text-sm font-medium text-fg shadow-md focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
      >
        Skip to content
      </a>
      <PublicHeader />
      <main id="main" className="flex-1">
        {children}
      </main>
      <PublicFooter />
    </div>
  );
}
