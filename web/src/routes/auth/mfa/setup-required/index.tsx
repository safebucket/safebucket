import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useLogout } from "@/hooks/useAuth";
import { MFASetupRequiredView } from "@/components/mfa-view/components/MFASetupRequiredView";
import { meQueryOptions } from "@/queries/me";
import { authProvidersQueryOptions } from "@/queries/auth_providers.ts";

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

  const { data: session } = useQuery(meQueryOptions());
  const { data: providers } = useQuery(authProvidersQueryOptions());
  const providerType = providers?.find(
    (p) => p.id === session?.authProvider,
  )?.type;

  return (
    <MFASetupRequiredView
      providerType={providerType}
      redirectPath={redirect}
      onLogout={logout}
    />
  );
}
