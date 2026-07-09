import { createFileRoute } from "@tanstack/react-router";
import { AdminSettingsOverview } from "@/components/admin-settings/AdminSettingsOverview";

export const Route = createFileRoute("/_authenticated/admin/settings/")({
  component: AdminSettingsOverview,
});
