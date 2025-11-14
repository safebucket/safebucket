import { useEffect } from "react";

import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { useRefreshSession, useSession } from "@/hooks/useAuth";
import { LoadingView } from "@/components/common/components/LoadingView.tsx";

export const Route = createFileRoute("/auth/complete/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: CompleteAuthComponent,
});

function CompleteAuthComponent() {
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const refreshSession = useRefreshSession();
  const session = useSession();

  useEffect(() => {
    // Refresh session to pick up new cookies set by OAuth
    refreshSession();

    // Navigate immediately if session exists
    if (session) {
      navigate({ to: redirect || "/", replace: true });
    }
  }, [session, redirect, navigate, refreshSession]);

  return <LoadingView />;
}
