import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { Showcase } from "./Showcase";

export const metadata: Metadata = { title: "UI kit", robots: { index: false, follow: false } };

// Development-only catalogue (production 404s it via next.config.ts rewrites) of design tokens and UI kit components (session 0.02).
export default function UiKitPage() {
  if (process.env.NODE_ENV === "production") notFound();
  return <Showcase />;
}
