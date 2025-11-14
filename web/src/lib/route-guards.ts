import { redirect } from "@tanstack/react-router";
import { isAuthenticated } from "@/lib/auth-service";

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
export function requireAuth({ location }: { location: { href: string } }) {
  if (!isAuthenticated()) {
    throw redirect({
      to: "/auth/login",
      search: {
        redirect: location.href,
      },
    });
  }
}
