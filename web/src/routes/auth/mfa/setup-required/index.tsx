import { createFileRoute } from "@tanstack/react-router";
import { useLogout } from "@/hooks/useAuth";
import { MFASetupRequiredView } from "@/components/mfa-view/components/MFASetupRequiredView";

export const Route = createFileRoute("/auth/mfa/setup-required/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: MFASetupRequired,
});

function MFASetupRequired() {
  const { redirect } = Route.useSearch();
  const logout = useLogout();

  return <MFASetupRequiredView redirectPath={redirect} onLogout={logout} />;
}
