import { createFileRoute } from "@tanstack/react-router";
import { AdminSettingsDetails } from "@/components/admin-settings/AdminSettingsDetails";

export const Route = createFileRoute("/_authenticated/admin/settings/details")({
  component: AdminSettingsDetails,
});
