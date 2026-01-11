import { Outlet, createFileRoute } from "@tanstack/react-router";
import { requireAuth } from "@/lib/route-guards";

/**
 * Protected layout route
 * All child routes under _authenticated require authentication
 */
export const Route = createFileRoute("/_authenticated")({
  beforeLoad: requireAuth,
  component: () => <Outlet />,
});
