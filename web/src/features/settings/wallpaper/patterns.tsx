import { useId } from "react";

import { cn } from "@/lib/cn";

import type { GalleryId } from "./gallery";

// Gallery wallpapers as inline SVG. Colours are theme tokens (fill-*/stroke-* utilities), so a
// pattern follows Light/Dark and the accent with no image files. Every shape is static: no motion.

const VIEW = { width: 1600, height: 1000 };

/** useId() output can contain characters that break url(#id) references; keep it to [A-Za-z0-9_-]. */
function useSvgId(prefix: string): string {
  return `${prefix}-${useId().replace(/[^A-Za-z0-9_-]/g, "")}`;
}

// Deterministic "topographic" rings: a circle with a few sine wobbles, sampled into a path.
function ring(cx: number, cy: number, r: number, seed: number): string {
  const points: string[] = [];
  const steps = 72;
  for (let i = 0; i <= steps; i++) {
    const a = (i / steps) * Math.PI * 2;
    const wobble = 1 + 0.08 * Math.sin(3 * a + seed) + 0.05 * Math.sin(5 * a + seed * 1.7);
    points.push(
      `${(cx + Math.cos(a) * r * wobble).toFixed(1)},${(cy + Math.sin(a) * r * 0.72 * wobble).toFixed(1)}`,
    );
  }
  return `M${points.join("L")}Z`;
}

const CONTOURS = [
  ...Array.from({ length: 9 }, (_, i) => ring(420, 320, 60 + i * 55, 0.4 + i * 0.3)),
  ...Array.from({ length: 7 }, (_, i) => ring(1250, 760, 50 + i * 60, 2.1 + i * 0.25)),
];

// Horizontal waves: y = base + amplitude · sin(x · frequency + phase), sampled every 20 px.
function wave(base: number, amplitude: number, frequency: number, phase: number): string {
  const points: string[] = [];
  for (let x = 0; x <= VIEW.width; x += 20) {
    points.push(`${x},${(base + amplitude * Math.sin(x * frequency + phase)).toFixed(1)}`);
  }
  return `M${points.join("L")}`;
}

const WAVES = Array.from({ length: 9 }, (_, i) =>
  wave(160 + i * 90, 40 + (i % 3) * 14, 0.004 + i * 0.0003, i * 0.7),
);

function Aurora() {
  const filter = useSvgId("aurora");
  return (
    <>
      <defs>
        <filter id={filter} x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="90" />
        </filter>
      </defs>
      <g filter={`url(#${filter})`}>
        <ellipse cx="300" cy="220" rx="420" ry="220" className="fill-chart-1 opacity-50" />
        <ellipse cx="1100" cy="180" rx="380" ry="200" className="fill-chart-4 opacity-40" />
        <ellipse cx="800" cy="760" rx="520" ry="230" className="fill-chart-2 opacity-40" />
        <ellipse cx="1450" cy="820" rx="300" ry="200" className="fill-primary opacity-30" />
      </g>
    </>
  );
}

function Dots() {
  const pattern = useSvgId("dots");
  return (
    <>
      <defs>
        <pattern id={pattern} width="32" height="32" patternUnits="userSpaceOnUse">
          <circle cx="16" cy="16" r="2.5" className="fill-primary" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill={`url(#${pattern})`} className="opacity-35" />
    </>
  );
}

function Grid() {
  const small = useSvgId("grid-s");
  const large = useSvgId("grid-l");
  return (
    <>
      <defs>
        <pattern id={small} width="40" height="40" patternUnits="userSpaceOnUse">
          <path d="M40 0H0V40" fill="none" strokeWidth="1" className="stroke-border-strong" />
        </pattern>
        <pattern id={large} width="200" height="200" patternUnits="userSpaceOnUse">
          <path d="M200 0H0V200" fill="none" strokeWidth="1.5" className="stroke-primary" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill={`url(#${small})`} />
      <rect width="100%" height="100%" fill={`url(#${large})`} className="opacity-30" />
    </>
  );
}

function Waves() {
  return (
    <g fill="none" strokeWidth="2" strokeLinecap="round">
      {WAVES.map((d, i) => (
        <path
          key={d}
          d={d}
          className={cn(
            i % 3 === 0 ? "stroke-primary" : i % 3 === 1 ? "stroke-chart-2" : "stroke-chart-1",
            "opacity-40",
          )}
        />
      ))}
    </g>
  );
}

function Contours() {
  return (
    <g fill="none" strokeWidth="1.5">
      {CONTOURS.map((d, i) => (
        <path
          key={d}
          d={d}
          className={cn("stroke-primary", i % 4 === 0 ? "opacity-60" : "opacity-30")}
        />
      ))}
    </g>
  );
}

function Peaks() {
  return (
    <>
      <path
        d="M0 640L220 470L420 590L640 380L860 560L1080 420L1300 540L1600 360V1000H0Z"
        className="fill-chart-1 opacity-20"
      />
      <path
        d="M0 760L260 600L500 720L760 540L1000 700L1240 580L1600 720V1000H0Z"
        className="fill-primary opacity-25"
      />
      <path
        d="M0 860L300 740L560 830L840 700L1120 820L1380 740L1600 820V1000H0Z"
        className="fill-chart-2 opacity-30"
      />
    </>
  );
}

const PATTERNS: Record<GalleryId, () => React.JSX.Element> = {
  aurora: Aurora,
  dots: Dots,
  grid: Grid,
  waves: Waves,
  contours: Contours,
  peaks: Peaks,
};

/** A gallery wallpaper filling its parent (cover, centred). Decorative: hidden from assistive tech. */
export function GalleryPattern({ id, className }: { id: GalleryId; className?: string }) {
  const Pattern = PATTERNS[id];
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox={`0 0 ${VIEW.width} ${VIEW.height}`}
      preserveAspectRatio="xMidYMid slice"
      className={cn("size-full", className)}
      data-testid={`gallery-pattern-${id}`}
    >
      <Pattern />
    </svg>
  );
}
