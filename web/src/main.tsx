import { StrictMode } from "react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import ReactDOM from "react-dom/client";

import reportWebVitals from "./reportWebVitals.ts";
import { routeTree } from "./routeTree.gen";
import { ThemeProvider } from "@/components/theme/context/ThemeProvider.tsx";
import { SidebarProvider } from "@/components/ui/sidebar.tsx";
import { UploadProvider } from "@/components/upload/context/UploadProvider.tsx";
import { MFAAuthProvider } from "@/context/MFAAuthContext.tsx";

import "./lib/i18n";
import "./styles.css";
import { getCurrentSessionWithRefresh } from "@/lib/auth-service.ts";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      retry: 1,
    },
  },
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

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Initialize app with session refresh check
async function initializeApp() {
  // Try to get session with automatic token refresh
  const session = await getCurrentSessionWithRefresh();

  // Update router context with refreshed session
  router.update({
    context: {
      queryClient,
      session,
    },
  });

  // Render the app
  const rootElement = document.getElementById("app");
  if (rootElement && !rootElement.innerHTML) {
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <StrictMode>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <SidebarProvider>
              <UploadProvider>
                <MFAAuthProvider>
                  <RouterProvider router={router} />
                </MFAAuthProvider>
              </UploadProvider>
            </SidebarProvider>
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
