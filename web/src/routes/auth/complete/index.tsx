import { useEffect } from "react";

import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
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
  const { status, refreshSession } = useSessionContext();

  useEffect(() => {
    // Refresh session to pick up new cookies set by OAuth
    refreshSession();
  }, [refreshSession]);

  useEffect(() => {
    if (status === "authenticated") {
      navigate({ to: redirect || "/", replace: true });
    }
  }, [status, redirect, navigate]);

  return <LoadingView />;
}
