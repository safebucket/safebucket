import { createFileRoute } from "@tanstack/react-router";
import { AdminActivityView } from "@/components/admin-activity/AdminActivityView";

export const Route = createFileRoute("/_authenticated/admin/activity/")({
  component: AdminActivityView,
});
