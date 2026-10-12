// Checks the color tokens in globals.css: every accent defines the full set in both themes, and
// text/background pairs meet WCAG AA contrast (PRD §4.1 "good contrast and accessibility").
import { readFileSync } from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { ACCENTS } from "@/lib/theme/theme";

const css = readFileSync(path.join(import.meta.dirname, "globals.css"), "utf8").replace(
  /\/\*[\s\S]*?\*\//g,
  "",
);

/** Innermost `selector { declarations }` blocks, keyed by each selector in the list. */
function blocksFor(selector: string): Record<string, string> {
  const vars: Record<string, string> = {};
  for (const match of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const selectors = (match[1] ?? "").split(",").map((item) => item.trim());
    if (!selectors.includes(selector)) continue;
    for (const decl of (match[2] ?? "").matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
      vars[decl[1] as string] = (decl[2] as string).trim();
    }
  }
  return vars;
}

function tokens(theme: "light" | "dark", accent: string): Record<string, string> {
  if (theme === "light") {
    return {
      ...blocksFor('[data-theme="light"]'),
      ...blocksFor(`[data-accent="${accent}"]`),
    };
  }
  return {
    ...blocksFor('[data-theme="dark"]'),
    ...blocksFor(`[data-theme="dark"][data-accent="${accent}"]`),
  };
}

/** Relative luminance of an opaque `oklch(L C H)` color (OKLab → linear sRGB). */
function luminance(value: string): number {
  const match = value.match(/^oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*\)$/);
  if (!match) throw new Error(`not an opaque oklch color: ${value}`);
  const [L, C, H] = [Number(match[1]), Number(match[2]), (Number(match[3]) * Math.PI) / 180];
  const a = C * Math.cos(H);
  const b = C * Math.sin(H);
  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3;
  const clamp = (x: number) => Math.min(1, Math.max(0, x));
  const r = clamp(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s);
  const g = clamp(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s);
  const bl = clamp(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s);
  return 0.2126 * r + 0.7152 * g + 0.0722 * bl;
}

function contrast(fg: string, bg: string): number {
  const [a, b] = [luminance(fg), luminance(bg)];
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

const PRIMARY_VARS = [
  "--primary",
  "--primary-hover",
  "--primary-fg",
  "--primary-soft",
  "--primary-text",
  "--ring",
];

// [text, background, minimum ratio]. 4.5 = AA body text, 3 = UI components / focus indicators.
const NEUTRAL_PAIRS: Array<[string, string, number]> = [
  ["--fg", "--canvas", 4.5],
  ["--fg", "--surface", 4.5],
  ["--fg-muted", "--surface", 4.5],
  ["--fg-muted", "--canvas", 4.5],
  ["--fg-muted", "--surface-muted", 4.5],
  ["--fg-subtle", "--surface", 4.5],
  ["--secondary-fg", "--secondary", 4.5],
  ["--success-text", "--success-soft", 4.5],
  ["--warning-text", "--warning-soft", 4.5],
  ["--error-text", "--error-soft", 4.5],
  ["--error-text", "--surface", 4.5],
  ["--info-text", "--info-soft", 4.5],
  ["--error-fg", "--error", 4.5],
];

const ACCENT_PAIRS: Array<[string, string, number]> = [
  ["--primary-fg", "--primary", 4.5],
  ["--primary-fg", "--primary-hover", 4.5],
  ["--primary-text", "--surface", 4.5],
  ["--primary-text", "--primary-soft", 4.5],
  ["--fg", "--primary-soft", 4.5],
  ["--ring", "--surface", 3],
  ["--primary", "--surface", 3],
];

describe.each(["light", "dark"] as const)("%s theme tokens", (theme) => {
  it.each(NEUTRAL_PAIRS)("%s on %s ≥ %s:1", (text, background, min) => {
    const vars = tokens(theme, "neutral");
    expect(contrast(vars[text] as string, vars[background] as string)).toBeGreaterThanOrEqual(min);
  });

  describe.each(ACCENTS)("accent %s", (accent) => {
    it("defines every primary token", () => {
      const own =
        theme === "light"
          ? blocksFor(`[data-accent="${accent}"]`)
          : blocksFor(`[data-theme="dark"][data-accent="${accent}"]`);
      expect(Object.keys(own).sort()).toEqual([...PRIMARY_VARS].sort());
    });

    it.each(ACCENT_PAIRS)("%s on %s ≥ %s:1", (text, background, min) => {
      const vars = tokens(theme, accent);
      expect(contrast(vars[text] as string, vars[background] as string)).toBeGreaterThanOrEqual(
        min,
      );
    });
  });
});
