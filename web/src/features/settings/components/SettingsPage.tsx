"use client";

import { useState } from "react";

import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui";

import type { SidebarState } from "@/features/learner-shell";

import { AppearanceSettings } from "./AppearanceSettings";
import { LearningPreferences } from "./LearningPreferences";
import { NotificationSettings } from "./NotificationSettings";

export type SettingsTab = "appearance" | "learning" | "notifications";

/** Learner Settings (PRD §20): Appearance, Learning preferences, Notifications. */
export function SettingsPage({ initialSidebar }: { initialSidebar: SidebarState }) {
  const [tab, setTab] = useState<SettingsTab>("appearance");
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight text-fg">Settings</h1>
        <p className="text-sm text-fg-muted">
          Make the platform look and work the way you study best.
        </p>
      </div>
      <Tabs value={tab} onValueChange={(v) => setTab(v as SettingsTab)}>
        <TabsList aria-label="Settings sections" className="overflow-x-auto">
          <TabsTrigger value="appearance">Appearance</TabsTrigger>
          <TabsTrigger value="learning">Learning preferences</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
        </TabsList>
        <TabsPanel value="appearance" className="pt-6">
          <AppearanceSettings initialSidebar={initialSidebar} />
        </TabsPanel>
        <TabsPanel value="learning" className="pt-6">
          <LearningPreferences />
        </TabsPanel>
        <TabsPanel value="notifications" className="pt-6">
          <NotificationSettings />
        </TabsPanel>
      </Tabs>
    </div>
  );
}
