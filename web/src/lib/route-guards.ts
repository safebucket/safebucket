import { redirect } from "@tanstack/react-router";
import type { QueryClient } from "@tanstack/react-query";

import type { Session } from "@/components/auth-view/types/session";
import { meQueryOptions } from "@/queries/me";

async function resolveSession(
  queryClient: QueryClient,
): Promise<Session | null> {
  return queryClient.ensureQueryData(meQueryOptions()).catch(() => null);
}

export async function requireAuth({
  location,
  context,
}: {
  location: { href: string };
  context: { queryClient: QueryClient; session: Session | null };
}) {
  const session = await resolveSession(context.queryClient);

  if (!session) {
    throw redirect({
      to: "/auth/login",
      search: {
        redirect: location.href,
      },
    });
  }
}

export async function requireAdmin({
  location,
  context,
}: {
  location: { href: string };
  context: { queryClient: QueryClient; session: Session | null };
}) {
  const session = await resolveSession(context.queryClient);

  if (!session) {
    throw redirect({
      to: "/auth/login",
      search: {
        redirect: location.href,
      },
    });
  }

  if (session.role !== "admin") {
    throw redirect({
      to: "/",
    });
  }
}
