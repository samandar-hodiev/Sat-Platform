import { themeScript } from "@/lib/theme/script";

/** Put in <head>: applies stored appearance before the first paint (no light-mode flash). */
export function ThemeScript() {
  return <script dangerouslySetInnerHTML={{ __html: themeScript() }} />;
}
