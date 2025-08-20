import { StrictMode } from "react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import ReactDOM from "react-dom/client";

import reportWebVitals from "./reportWebVitals.ts";
import { routeTree } from "./routeTree.gen";
import { SessionProvider } from "@/components/auth-view/context/SessionProvider.tsx";
import { SidebarProvider } from "@/components/ui/sidebar.tsx";
import { UploadProvider } from "@/components/upload/context/UploadProvider.tsx";

import "./styles.css";

// Create a query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      retry: 1,
    },
  },
});

// Create a new router instance
export const router = createRouter({
  routeTree,
  context: {
    queryClient,
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

// Render the app
const rootElement = document.getElementById("app");
if (rootElement && !rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <SessionProvider>
          <SidebarProvider>
            <UploadProvider>
              <RouterProvider router={router} />
            </UploadProvider>
          </SidebarProvider>
        </SessionProvider>
      </QueryClientProvider>
    </StrictMode>,
  );
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
