import { createFileRoute } from "@tanstack/react-router";
import { AdminUsersView } from "@/components/admin-users/AdminUsersView";

export const Route = createFileRoute("/_authenticated/admin/users/")({
  component: AdminUsersView,
});
