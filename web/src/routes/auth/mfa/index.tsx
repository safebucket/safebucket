import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMFAAuth } from "@/context/MFAAuthContext";
import { MFAVerificationView } from "@/components/mfa-view/components/MFAVerificationView";

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
  const { mfaToken, devices } = useMFAAuth();

  if (!mfaToken) {
    navigate({ to: "/auth/login", search: { redirect } });
    return null;
  }

  return (
    <MFAVerificationView
      mfaToken={mfaToken}
      devices={devices}
      redirectPath={redirect}
    />
  );
}
