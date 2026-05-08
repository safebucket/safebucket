import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { MFAVerificationView } from "@/components/mfa-view/components/MFAVerificationView";
import { MFASetupRequiredView } from "@/components/mfa-view/components/MFASetupRequiredView";
import { mfaPending } from "@/components/mfa-view/helpers/token";
import { mfaDevicesQueryOptions } from "@/queries/mfa";

export const Route = createFileRoute("/auth/mfa/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  beforeLoad: ({ search }) => {
    if (!mfaPending.get()) {
      throw redirect({
        to: "/auth/login",
        search: { redirect: search.redirect },
      });
    }
  },
  component: MFAVerification,
});

function MFAVerification() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { redirect: redirectPath } = Route.useSearch();

  const { data, isLoading } = useQuery(mfaDevicesQueryOptions());
  const devices = data?.devices ?? [];

  const clearAuth = () => {
    mfaPending.clear();
    queryClient.removeQueries({ queryKey: ["mfa", "devices"] });
  };

  const handleLogout = () => {
    clearAuth();
    navigate({ to: "/auth/login", search: { redirect: undefined } });
  };

  if (isLoading) {
    return (
      <div className="m-6 flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (devices.length === 0) {
    return (
      <MFASetupRequiredView
        isRestricted
        redirectPath={redirectPath}
        onLogout={handleLogout}
      />
    );
  }

  return (
    <MFAVerificationView
      devices={devices}
      redirectPath={redirectPath}
      onClearAuth={clearAuth}
    />
  );
}
