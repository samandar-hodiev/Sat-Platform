import type { ReactNode } from "react";

import { Badge, CheckIcon } from "@/components/ui";
import { cn } from "@/lib/cn";

import type { Spotlight } from "../content";
import { FeatureIconGlyph } from "../icons";

// Schematic product previews for the public site. They show the shape of the UI, never data:
// no scores, percentages, counts or names, and every frame says it is an illustration
// (CLAUDE.md: demo data is never presented as real).

const LABELS: Record<Spotlight["visual"] | "dashboard", string> = {
  dashboard: "Illustration of the learner dashboard",
  practice: "Illustration of a practice question with answer choices",
  exam: "Illustration of the timed mock exam screen",
  assistant: "Illustration of a conversation with the AI study assistant",
  plan: "Illustration of a daily study plan",
  progress: "Illustration of progress by SAT domain",
};

function Frame({
  label,
  title,
  children,
  className,
}: {
  label: string;
  title: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <figure
      role="img"
      aria-label={label}
      className={cn("rounded-xl border border-border bg-surface shadow-md", className)}
    >
      <div aria-hidden className="flex items-center gap-2 border-b border-border px-4 py-3">
        <span className="size-2.5 rounded-full bg-border-strong" />
        <span className="size-2.5 rounded-full bg-border-strong" />
        <span className="size-2.5 rounded-full bg-border-strong" />
        <span className="ml-2 flex-1 truncate text-xs font-medium text-fg-muted">{title}</span>
        <Badge>Illustration</Badge>
      </div>
      <div aria-hidden className="p-4 sm:p-5">
        {children}
      </div>
    </figure>
  );
}

/** A placeholder line of text. */
function Line({ className }: { className?: string }) {
  return <span className={cn("block h-2 rounded-full bg-surface-muted", className)} />;
}

const DOMAINS = [
  { name: "Algebra", bar: "w-3/4 bg-chart-1" },
  { name: "Advanced Math", bar: "w-1/2 bg-chart-2" },
  { name: "Craft & Structure", bar: "w-2/3 bg-chart-3" },
  { name: "Expression of Ideas", bar: "w-2/5 bg-chart-4" },
];

const PLAN_TASKS = [
  { kind: "Lesson", title: "Linear equations in one variable", done: true },
  { kind: "Practice", title: "Words in context", done: true },
  { kind: "Review", title: "Questions from your mistake notebook", done: false },
  { kind: "Practice", title: "Systems of linear equations", done: false },
];

function PlanList({ items = PLAN_TASKS }: { items?: typeof PLAN_TASKS }) {
  return (
    <ul className="flex flex-col gap-2">
      {items.map((task) => (
        <li
          key={task.title}
          className="flex items-center gap-3 rounded-lg border border-border bg-canvas px-3 py-2.5"
        >
          <span
            className={cn(
              "flex size-5 shrink-0 items-center justify-center rounded-full border",
              task.done
                ? "border-transparent bg-success-soft text-success-text"
                : "border-border-strong",
            )}
          >
            {task.done ? <CheckIcon size={12} /> : null}
          </span>
          <span className="min-w-0 flex-1 truncate text-sm text-fg">{task.title}</span>
          <Badge
            variant={
              task.kind === "Lesson" ? "info" : task.kind === "Review" ? "warning" : "primary"
            }
          >
            {task.kind}
          </Badge>
        </li>
      ))}
    </ul>
  );
}

function DomainBars() {
  return (
    <ul className="flex flex-col gap-3">
      {DOMAINS.map((domain) => (
        <li key={domain.name} className="flex flex-col gap-1.5">
          <span className="text-xs font-medium text-fg-muted">{domain.name}</span>
          <span className="block h-2 rounded-full bg-surface-muted">
            <span className={cn("block h-2 rounded-full", domain.bar)} />
          </span>
        </li>
      ))}
    </ul>
  );
}

export function DashboardIllustration({ className }: { className?: string }) {
  return (
    <Frame label={LABELS.dashboard} title="Dashboard" className={className}>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-5">
        <div className="flex flex-col gap-3 sm:col-span-3">
          <p className="text-sm font-semibold text-fg">Today&apos;s plan</p>
          <PlanList items={PLAN_TASKS.slice(0, 3)} />
        </div>
        <div className="flex flex-col gap-3 rounded-lg border border-border bg-canvas p-4 sm:col-span-2">
          <p className="text-sm font-semibold text-fg">Progress by domain</p>
          <DomainBars />
        </div>
      </div>
    </Frame>
  );
}

const CHOICES = ["A", "B", "C", "D"];

function PracticeIllustration() {
  return (
    <Frame label={LABELS.practice} title="Daily Practice">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <Badge variant="info">Algebra</Badge>
          <span className="text-xs text-fg-subtle">Practice mode</span>
        </div>
        <div className="flex flex-col gap-2">
          <Line className="w-full" />
          <Line className="w-11/12" />
          <Line className="w-2/3" />
        </div>
        <ul className="flex flex-col gap-2">
          {CHOICES.map((choice) => (
            <li
              key={choice}
              className={cn(
                "flex items-center gap-3 rounded-lg border px-3 py-2.5",
                choice === "C"
                  ? "border-transparent bg-success-soft text-success-text"
                  : "border-border bg-canvas",
              )}
            >
              <span className="flex size-6 items-center justify-center rounded-full border border-current text-xs font-semibold">
                {choice}
              </span>
              <Line className={choice === "C" ? "w-1/2 bg-success/30" : "w-1/2"} />
              {choice === "C" ? <CheckIcon size={14} className="ml-auto" /> : null}
            </li>
          ))}
        </ul>
        <div className="rounded-lg bg-surface-muted p-3">
          <p className="text-xs font-semibold text-fg">Explanation</p>
          <div className="mt-2 flex flex-col gap-2">
            <Line className="w-full bg-border" />
            <Line className="w-3/4 bg-border" />
          </div>
        </div>
      </div>
    </Frame>
  );
}

function ExamIllustration() {
  return (
    <Frame label={LABELS.exam} title="Mock Exam">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between rounded-lg border border-border bg-canvas px-3 py-2">
          <span className="text-xs font-medium text-fg-muted">Section 2 · Math · Module 1</span>
          <span className="flex items-center gap-1.5 text-xs font-semibold text-fg">
            <FeatureIconGlyph name="timer" size={14} />
            Time remaining
          </span>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-2">
            <Line className="w-full" />
            <Line className="w-5/6" />
            <Line className="w-full" />
            <Line className="w-1/2" />
          </div>
          <div className="flex flex-col gap-2">
            {CHOICES.map((choice) => (
              <span
                key={choice}
                className={cn(
                  "flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs",
                  choice === "B"
                    ? "border-primary bg-primary-soft text-primary-text"
                    : "border-border",
                )}
              >
                <span className="font-semibold">{choice}</span>
                <Line className="w-2/3" />
              </span>
            ))}
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {Array.from({ length: 12 }, (_, index) => (
            <span
              key={index}
              className={cn(
                "size-6 rounded-sm border",
                index < 5 ? "border-transparent bg-primary-soft" : "border-border",
              )}
            />
          ))}
        </div>
      </div>
    </Frame>
  );
}

function AssistantIllustration() {
  return (
    <Frame label={LABELS.assistant} title="AI Study Assistant">
      <div className="flex flex-col gap-3">
        <div className="ml-auto max-w-xs rounded-lg bg-primary px-3 py-2.5 text-sm text-primary-fg">
          Why is answer B wrong here?
        </div>
        <div className="flex max-w-sm gap-2">
          <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-text">
            <FeatureIconGlyph name="sparkles" size={14} />
          </span>
          <div className="flex flex-1 flex-col gap-2 rounded-lg bg-surface-muted p-3">
            <p className="text-xs font-semibold text-fg">Step 1</p>
            <Line className="w-full bg-border" />
            <Line className="w-4/5 bg-border" />
            <p className="text-xs font-semibold text-fg">Step 2</p>
            <Line className="w-11/12 bg-border" />
            <Line className="w-2/3 bg-border" />
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-lg border border-border bg-canvas px-3 py-2.5">
          <span className="flex-1 text-sm text-fg-subtle">Ask about a concept…</span>
        </div>
      </div>
    </Frame>
  );
}

function PlanIllustration() {
  return (
    <Frame label={LABELS.plan} title="My Study Plan">
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-7 gap-1.5">
          {["M", "T", "W", "T", "F", "S", "S"].map((day, index) => (
            <span
              key={`${day}-${index}`}
              className={cn(
                "flex h-9 items-center justify-center rounded-md text-xs font-medium",
                index === 2 ? "bg-primary text-primary-fg" : "bg-surface-muted text-fg-muted",
              )}
            >
              {day}
            </span>
          ))}
        </div>
        <PlanList />
      </div>
    </Frame>
  );
}

function ProgressIllustration() {
  return (
    <Frame label={LABELS.progress} title="My Progress">
      <div className="flex flex-col gap-5">
        <div className="flex h-28 items-end gap-2 border-b border-chart-grid">
          {["h-1/4", "h-2/5", "h-1/3", "h-1/2", "h-3/5", "h-2/3", "h-3/4"].map((height, index) => (
            <span key={index} className={cn("flex-1 rounded-t-sm bg-chart-1", height)} />
          ))}
        </div>
        <DomainBars />
      </div>
    </Frame>
  );
}

export function SpotlightIllustration({ visual }: { visual: Spotlight["visual"] }) {
  switch (visual) {
    case "practice":
      return <PracticeIllustration />;
    case "exam":
      return <ExamIllustration />;
    case "assistant":
      return <AssistantIllustration />;
    case "plan":
      return <PlanIllustration />;
    case "progress":
      return <ProgressIllustration />;
  }
}
