import { redirect } from "@tanstack/react-router";

/**
 * Route guard to require authentication
 * Use in route's beforeLoad to protect the route
 *
 * @example
 * export const Route = createFileRoute('/protected')({
 *   beforeLoad: requireAuth,
 *   component: ProtectedComponent
 * })
 */
export function requireAuth({
  location,
  context,
}: {
  location: { href: string };
  context: { session: any };
}) {
  // Check session from router context (single source of truth)
  if (!context.session) {
    throw redirect({
      to: "/auth/login",
      search: {
        redirect: location.href,
      },
    });
  }
}

/**
 * Route guard to require admin role
 * Use in route's beforeLoad to protect admin-only routes
 *
 * @example
 * export const Route = createFileRoute('/admin')({
 *   beforeLoad: requireAdmin,
 *   component: AdminComponent
 * })
 */
export function requireAdmin({
  location,
  context,
}: {
  location: { href: string };
  context: { session: any };
}) {
  requireAuth({ location, context });

  if (context.session.role !== "admin") {
    throw redirect({
      to: "/",
    });
  }
}
