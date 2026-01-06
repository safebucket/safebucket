import { createFileRoute } from "@tanstack/react-router";
import { AdminDashboard } from "@/components/admin-dashboard/AdminDashboard";

export const Route = createFileRoute("/_authenticated/admin/dashboard/")({
  component: AdminDashboard,
});
