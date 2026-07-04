import { StrictMode } from "react";

import {
  MutationCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import ReactDOM from "react-dom/client";

import reportWebVitals from "./reportWebVitals.ts";
import { routeTree } from "./routeTree.gen";
import { errorToast } from "@/components/ui/hooks/use-toast.ts";
import { ThemeProvider } from "@/components/theme/context/ThemeProvider.tsx";
import { TimeDisplayProvider } from "@/components/time-display/context/TimeDisplayProvider.tsx";
import { SidebarProvider } from "@/components/ui/sidebar.tsx";
import { UploadProvider } from "@/components/upload/context/UploadProvider.tsx";

import "./lib/i18n";
import "./styles.css";
import { getCurrentSessionWithRefresh } from "@/lib/auth-service.ts";
import { configQueryOptions } from "@/queries/config.ts";

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: {
      skipGlobalErrorToast?: boolean;
    };
  }
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      if (mutation.options.meta?.skipGlobalErrorToast) return;
      errorToast(error);
    },
  }),
});

export const router = createRouter({
  routeTree,
  context: {
    queryClient,
    session: null,
  },
  defaultPreload: "intent",
  scrollRestoration: true,
  defaultStructuralSharing: true,
  defaultPreloadStaleTime: 0,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

async function initializeApp() {
  await queryClient.ensureQueryData(configQueryOptions());
  const session = await getCurrentSessionWithRefresh();

  router.update({
    context: {
      queryClient,
      session,
    },
  });

  const rootElement = document.getElementById("app");
  if (rootElement && !rootElement.innerHTML) {
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <StrictMode>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <TimeDisplayProvider>
              <SidebarProvider>
                <UploadProvider>
                  <RouterProvider router={router} />
                </UploadProvider>
              </SidebarProvider>
            </TimeDisplayProvider>
          </ThemeProvider>
        </QueryClientProvider>
      </StrictMode>,
    );
  }
}

initializeApp();

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
