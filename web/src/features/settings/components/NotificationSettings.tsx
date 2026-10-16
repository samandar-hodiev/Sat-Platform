import { EmptyState } from "@/components/ui";

/**
 * Settings → Notifications (PRD §20, §31). Reminder and summary switches need the notification
 * types and delivery that session 3.14 builds; until then this tab says so instead of showing
 * switches that would do nothing.
 */
export function NotificationSettings() {
  return (
    <EmptyState
      title="Notification settings are not available yet"
      description="Study reminders, plan reminders and the weekly summary will be set up here once notifications are part of the platform."
    />
  );
}
