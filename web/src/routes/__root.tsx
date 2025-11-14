import { TanStackDevtools } from "@tanstack/react-devtools";
import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import type { QueryClient } from "@tanstack/react-query";

import type { Session } from "@/components/auth-view/types/session";
import { AppSidebar } from "@/components/app-sidebar/AppSidebar.tsx";
import { AppSidebarInset } from "@/components/app-sidebar/components/AppSidebarInset.tsx";
import { Toaster } from "@/components/ui/toaster.tsx";
import { EnvironmentType } from "@/types/app.ts";
import { useConfig } from "@/hooks/useConfig.ts";
import { configQueryOptions } from "@/queries/config.ts";

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
  session: Session | null;
}>()({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(configQueryOptions()),
  component: RootComponent,
});

function RootComponent() {
  const { session } = Route.useRouteContext();
  const config = useConfig();

  return (
    <div className="flex h-svh max-h-svh w-full">
      {session && <AppSidebar />}
      <AppSidebarInset>
        <Outlet />
      </AppSidebarInset>
      <Toaster />

      {config.environment == EnvironmentType.development && (
        <TanStackDevtools
          config={{
            position: "bottom-left",
          }}
          plugins={[
            {
              name: "TanStack Router",
              render: <TanStackRouterDevtoolsPanel />,
            },
          ]}
        />
      )}
    </div>
  );
}
