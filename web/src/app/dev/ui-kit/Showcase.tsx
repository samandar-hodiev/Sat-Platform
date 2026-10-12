"use client";

import { useState, type ReactNode } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";
import { ThemeSwitcher } from "@/components/theme/ThemeSwitcher";
import {
  Accordion,
  AccordionItem,
  Avatar,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Checkbox,
  ChoiceGroup,
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  EmptyState,
  ErrorState,
  FormField,
  IconButton,
  Input,
  LoadingState,
  Modal,
  MoreHorizontalIcon,
  Radio,
  RefreshIcon,
  SearchIcon,
  Select,
  Skeleton,
  SkeletonText,
  Spinner,
  Switch,
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
  Textarea,
  Tooltip,
  useToast,
} from "@/components/ui";
import { cn } from "@/lib/cn";
import { ACCENTS, DENSITIES } from "@/lib/theme/theme";

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-sm font-semibold tracking-wide text-fg-subtle uppercase">{title}</h2>
      {children}
    </section>
  );
}

const colorTokens = [
  "canvas",
  "surface",
  "surface-muted",
  "fg",
  "fg-muted",
  "border-strong",
  "primary",
  "primary-soft",
  "secondary",
  "success",
  "warning",
  "error",
  "info",
  "chart-1",
  "chart-2",
  "chart-3",
  "chart-4",
  "chart-5",
] as const;

export function Showcase() {
  const theme = useTheme();
  const { toast } = useToast();
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-12 px-4 py-10 sm:px-6">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="flex flex-col gap-1">
          <h1 className="text-3xl font-semibold tracking-tight">UI kit</h1>
          <p className="text-fg-muted">Design tokens and components. Development only.</p>
        </div>
        <ThemeSwitcher showLabels />
      </header>

      <Section title="Appearance">
        <Card>
          <CardContent className="flex flex-col gap-5">
            <div className="flex flex-col gap-2">
              <span className="text-sm font-medium">Accent</span>
              <div role="radiogroup" aria-label="Accent color" className="flex flex-wrap gap-2">
                {ACCENTS.map((accent) => (
                  <button
                    key={accent}
                    type="button"
                    role="radio"
                    aria-checked={theme.accent === accent}
                    aria-label={accent}
                    data-accent={accent}
                    onClick={() => theme.setAccent(accent)}
                    className={cn(
                      "focus-ring size-8 rounded-full border-2 bg-primary",
                      theme.accent === accent ? "border-fg" : "border-transparent",
                    )}
                  />
                ))}
              </div>
            </div>
            <div className="flex flex-wrap gap-6">
              <Select
                aria-label="Density"
                value={theme.density}
                onChange={(event) =>
                  theme.setDensity(event.target.value as (typeof DENSITIES)[number])
                }
                className="w-44"
              >
                {DENSITIES.map((density) => (
                  <option key={density} value={density}>
                    {density}
                  </option>
                ))}
              </Select>
              <Switch
                label="Reduce motion"
                checked={theme.motion === "reduced"}
                onCheckedChange={(on) => theme.setMotion(on ? "reduced" : "system")}
              />
            </div>
            <p className="text-sm text-fg-muted">
              Scope: <strong>{theme.scope}</strong> · resolved theme:{" "}
              <strong>{theme.resolvedTheme}</strong>
            </p>
          </CardContent>
        </Card>
      </Section>

      <Section title="Color tokens">
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
          {colorTokens.map((token) => (
            <div key={token} className="flex flex-col gap-1.5">
              <div
                className="h-12 rounded-md border border-border"
                style={{ backgroundColor: `var(--${token})` }}
              />
              <code className="text-xs text-fg-muted">{token}</code>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Typography">
        <div className="flex flex-col gap-2">
          <p className="text-4xl font-semibold tracking-tight">Heading 4xl</p>
          <p className="text-3xl font-semibold tracking-tight">Heading 3xl</p>
          <p className="text-2xl font-semibold">Heading 2xl</p>
          <p className="text-xl font-semibold">Heading xl</p>
          <p className="text-lg font-medium">Title lg</p>
          <p className="text-base">Body base — the quick brown fox jumps over the lazy dog.</p>
          <p className="text-sm text-fg-muted">Small muted — supporting text and hints.</p>
          <p className="text-xs text-fg-subtle">Extra small subtle — captions and metadata.</p>
        </div>
      </Section>

      <Section title="Buttons">
        <div className="flex flex-wrap items-center gap-3">
          <Button>Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="danger">Delete</Button>
          <Button disabled>Disabled</Button>
          <Button
            loading={saving}
            onClick={() => {
              setSaving(true);
              setTimeout(() => setSaving(false), 1500);
            }}
          >
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
          <Tooltip content="Search">
            <IconButton aria-label="Search" variant="outline">
              <SearchIcon />
            </IconButton>
          </Tooltip>
          <Tooltip content="Refresh data" side="bottom">
            <IconButton aria-label="Refresh">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </div>
      </Section>

      <Section title="Form controls">
        <Card>
          <CardContent className="grid gap-5 sm:grid-cols-2">
            <FormField label="Full name" hint="As it should appear on your profile." required>
              <Input placeholder="e.g. Aziz Karimov" />
            </FormField>
            <FormField label="Email" error="Enter a valid email address.">
              <Input type="email" defaultValue="aziz@" />
            </FormField>
            <FormField label="Target score">
              <Select defaultValue="" placeholder="Choose a target">
                <option value="1200">1200</option>
                <option value="1400">1400</option>
                <option value="1500">1500+</option>
              </Select>
            </FormField>
            <FormField label="Disabled">
              <Input disabled value="Not editable" />
            </FormField>
            <FormField label="Notes" className="sm:col-span-2">
              <Textarea placeholder="Anything we should know?" />
            </FormField>
            <ChoiceGroup legend="Study time per day">
              <Radio name="time" value="30" label="30 minutes" defaultChecked />
              <Radio
                name="time"
                value="60"
                label="1 hour"
                description="Recommended for most learners."
              />
              <Radio name="time" value="120" label="2 hours" disabled />
            </ChoiceGroup>
            <div className="flex flex-col gap-4">
              <Checkbox
                label="Email reminders"
                description="One short email per study day."
                defaultChecked
              />
              <Checkbox label="Weekly report" />
              <Switch
                label="Show timer"
                description="Visible during practice sets."
                defaultChecked
              />
            </div>
          </CardContent>
        </Card>
      </Section>

      <Section title="Badges and avatars">
        <div className="flex flex-wrap items-center gap-2">
          <Badge>Draft</Badge>
          <Badge variant="primary">New</Badge>
          <Badge variant="success">Published</Badge>
          <Badge variant="warning">In review</Badge>
          <Badge variant="error">Failed</Badge>
          <Badge variant="info">Info</Badge>
        </div>
        <div className="flex items-center gap-3">
          <Avatar name="Aziz Karimov" size="sm" />
          <Avatar name="Madina" />
          <Avatar name="Owner Admin" size="lg" />
        </div>
      </Section>

      <Section title="Cards">
        <div className="grid gap-4 sm:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Algebra · Linear equations</CardTitle>
              <CardDescription>12 questions · about 15 minutes</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-fg-muted">
              Practice solving linear equations in one variable.
            </CardContent>
            <CardFooter>
              <Button size="sm">Start</Button>
              <Button size="sm" variant="ghost">
                Preview
              </Button>
            </CardFooter>
          </Card>
          <Card aria-busy="true">
            <CardHeader>
              <Skeleton className="h-5 w-2/3" />
              <Skeleton className="h-4 w-1/3" />
            </CardHeader>
            <CardContent>
              <SkeletonText lines={3} />
            </CardContent>
          </Card>
        </div>
      </Section>

      <Section title="Tabs and accordion">
        <Tabs defaultValue="daily">
          <TabsList aria-label="Plan view">
            <TabsTrigger value="daily">Daily</TabsTrigger>
            <TabsTrigger value="weekly">Weekly</TabsTrigger>
            <TabsTrigger value="archive" disabled>
              Archive
            </TabsTrigger>
          </TabsList>
          <TabsPanel value="daily" className="text-sm text-fg-muted">
            Today’s tasks appear here.
          </TabsPanel>
          <TabsPanel value="weekly" className="text-sm text-fg-muted">
            This week’s plan appears here.
          </TabsPanel>
        </Tabs>
        <Accordion defaultValue={["math"]}>
          <AccordionItem value="math" title="Math" meta={<Badge variant="success">4 topics</Badge>}>
            Algebra, Advanced Math, Problem-Solving and Data Analysis, Geometry and Trigonometry.
          </AccordionItem>
          <AccordionItem value="rw" title="Reading and Writing" meta={<Badge>4 topics</Badge>}>
            Information and Ideas, Craft and Structure, Expression of Ideas, Standard English
            Conventions.
          </AccordionItem>
        </Accordion>
      </Section>

      <Section title="Overlays">
        <div className="flex flex-wrap items-center gap-3">
          <Button variant="outline" onClick={() => setModalOpen(true)}>
            Open modal
          </Button>
          <DropdownMenu
            trigger={(props) => (
              <IconButton {...props} aria-label="More actions" variant="outline">
                <MoreHorizontalIcon />
              </IconButton>
            )}
          >
            <DropdownMenuLabel>Question</DropdownMenuLabel>
            <DropdownMenuItem onSelect={() => toast({ title: "Edit selected" })}>
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => toast({ title: "Duplicated", variant: "success" })}>
              Duplicate
            </DropdownMenuItem>
            <DropdownMenuItem disabled>Move (no permission)</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              destructive
              onSelect={() => toast({ title: "Archived", variant: "warning" })}
            >
              Archive
            </DropdownMenuItem>
          </DropdownMenu>
          <Button
            variant="ghost"
            onClick={() =>
              toast({ title: "Saved", description: "Your changes were saved.", variant: "success" })
            }
          >
            Success toast
          </Button>
          <Button
            variant="ghost"
            onClick={() =>
              toast({
                title: "Could not save",
                description: "Check your connection and try again.",
                variant: "error",
              })
            }
          >
            Error toast
          </Button>
          <Button variant="ghost" onClick={() => toast({ title: "Plan updated", variant: "info" })}>
            Info toast
          </Button>
        </div>
        <Modal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          title="Leave the practice set?"
          description="Your answers so far are saved. You can continue later."
          footer={
            <>
              <Button variant="outline" onClick={() => setModalOpen(false)}>
                Stay
              </Button>
              <Button onClick={() => setModalOpen(false)}>Leave</Button>
            </>
          }
        />
      </Section>

      <Section title="States">
        <div className="grid gap-4 sm:grid-cols-3">
          <Card>
            <LoadingState />
          </Card>
          <Card>
            <EmptyState
              title="No saved questions yet"
              description="Bookmark a question during practice to find it here."
              action={<Button size="sm">Start practice</Button>}
            />
          </Card>
          <Card>
            <ErrorState onRetry={() => toast({ title: "Retrying…" })} reference="req_8f2c1a" />
          </Card>
        </div>
        <div className="flex items-center gap-3 text-fg-muted">
          <Spinner size="sm" />
          <Spinner />
          <Spinner size="lg" />
        </div>
      </Section>
    </main>
  );
}
