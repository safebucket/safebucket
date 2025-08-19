import { TanstackDevtools } from "@tanstack/react-devtools";
import type { QueryClient } from "@tanstack/react-query";
import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";

import { AppSidebar } from "@/components/app-sidebar/AppSidebar.tsx";
import { AppSidebarInset } from "@/components/app-sidebar/components/AppSidebarInset.tsx";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext.ts";
import { Toaster } from "@/components/ui/toaster.tsx";

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  component: RootComponent,
});

function RootComponent() {
  const { session } = useSessionContext();

  return (
    <div className="flex h-svh max-h-svh w-full">
      {session && <AppSidebar />}
      <AppSidebarInset>
        <Outlet />
      </AppSidebarInset>
      <Toaster />

      <TanstackDevtools
        config={{
          position: "bottom-left",
        }}
        plugins={[
          {
            name: "Tanstack Router",
            render: <TanStackRouterDevtoolsPanel />,
          },
        ]}
      />
    </div>
  );
}
