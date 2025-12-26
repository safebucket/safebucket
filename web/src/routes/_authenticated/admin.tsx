import { Outlet, createFileRoute } from "@tanstack/react-router";
import { requireAdmin } from "@/lib/route-guards";

/**
 * Admin layout route
 * All child routes under admin require admin role
 */
export const Route = createFileRoute("/_authenticated/admin")({
  beforeLoad: requireAdmin,
  component: () => <Outlet />,
});
