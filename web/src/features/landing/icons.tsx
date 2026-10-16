import type { SVGProps } from "react";

import type { FeatureIcon } from "./content";

// Public site icons, drawn like the UI kit set (24×24, 1.75 stroke, currentColor). Decorative:
// every place they appear also has a text label or aria-label.
export type LandingIconProps = SVGProps<SVGSVGElement> & { size?: number };

function Icon({ size = 20, children, ...props }: LandingIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      focusable="false"
      {...props}
    >
      {children}
    </svg>
  );
}

export const MenuIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M4 6h16M4 12h16M4 18h16" />
  </Icon>
);

export const ArrowRightIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M5 12h14M13 6l6 6-6 6" />
  </Icon>
);

export const MailIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <rect x="3" y="5" width="18" height="14" rx="2" />
    <path d="m3 7 9 6 9-6" />
  </Icon>
);

const TargetIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <circle cx="12" cy="12" r="9" />
    <circle cx="12" cy="12" r="5" />
    <circle cx="12" cy="12" r="1" />
  </Icon>
);

const CalendarIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <rect x="3" y="5" width="18" height="16" rx="2" />
    <path d="M16 3v4M8 3v4M3 10h18M8 14h.01M12 14h.01M16 14h.01M8 17h.01M12 17h.01" />
  </Icon>
);

const BookIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5Z" />
    <path d="M4 19a2 2 0 0 1 2-2h13" />
  </Icon>
);

const PencilIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M4 20h4L19 9a2.8 2.8 0 0 0-4-4L4 16v4Z" />
    <path d="m13.5 6.5 4 4" />
  </Icon>
);

const TimerIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <circle cx="12" cy="13" r="8" />
    <path d="M12 9v4l2 2M10 2h4" />
  </Icon>
);

const NotebookIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <rect x="5" y="3" width="15" height="18" rx="2" />
    <path d="M9 3v18M3 7h4M3 12h4M3 17h4M13 8h4M13 12h4" />
  </Icon>
);

const ChartIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M4 20V4M4 20h16" />
    <path d="m8 15 4-4 3 3 5-6" />
  </Icon>
);

const SparklesIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M12 3l1.8 4.7L18.5 9.5l-4.7 1.8L12 16l-1.8-4.7L5.5 9.5l4.7-1.8L12 3Z" />
    <path d="M19 15l.8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8L19 15Z" />
  </Icon>
);

const FormulaIcon = (p: LandingIconProps) => (
  <Icon {...p}>
    <path d="M17 5H8l5 7-5 7h9" />
  </Icon>
);

const FEATURE_ICONS: Record<FeatureIcon, (p: LandingIconProps) => React.JSX.Element> = {
  target: TargetIcon,
  calendar: CalendarIcon,
  book: BookIcon,
  pencil: PencilIcon,
  timer: TimerIcon,
  notebook: NotebookIcon,
  chart: ChartIcon,
  sparkles: SparklesIcon,
  formula: FormulaIcon,
};

export function FeatureIconGlyph({ name, ...props }: LandingIconProps & { name: FeatureIcon }) {
  const Glyph = FEATURE_ICONS[name];
  return <Glyph {...props} />;
}
