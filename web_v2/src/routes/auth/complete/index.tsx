import { useEffect } from "react";

import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { LoadingView } from "@/components/common/components/LoadingView.tsx";

export const Route = createFileRoute("/auth/complete/")({
  component: CompleteAuthComponent,
});

function CompleteAuthComponent() {
  const navigate = useNavigate();
  const { status } = useSessionContext();

  useEffect(() => {
    if (status == "authenticated") {
      navigate({ to: "/", replace: true });
    }
  }, [status]);

  return <LoadingView />;
}
