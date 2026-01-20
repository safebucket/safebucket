import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useMFAAuth } from "@/context/MFAAuthContext";
import { MFAVerificationView } from "@/components/mfa-view/components/MFAVerificationView";
import { MFASetupRequiredView } from "@/components/mfa-view/components/MFASetupRequiredView";

export const Route = createFileRoute("/auth/mfa/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: MFAVerification,
});

function MFAVerification() {
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const { restrictedToken, devices, isLoadingDevices, clearMFAAuth } =
    useMFAAuth();

  const handleLogout = () => {
    clearMFAAuth();
    navigate({ to: "/auth/login", search: { redirect: undefined } });
  };

  if (!restrictedToken) {
    navigate({ to: "/auth/login", search: { redirect } });
    return null;
  }

  if (isLoadingDevices) {
    return (
      <div className="m-6 flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // No devices - show setup flow (user needs to enroll a device)
  if (devices.length === 0) {
    return (
      <MFASetupRequiredView redirectPath={redirect} onLogout={handleLogout} />
    );
  }

  return (
    <MFAVerificationView
      mfaToken={restrictedToken}
      devices={devices}
      redirectPath={redirect}
    />
  );
}
