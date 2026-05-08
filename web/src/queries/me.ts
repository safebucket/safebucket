import { queryOptions } from "@tanstack/react-query";

import type { Session } from "@/components/auth-view/types/session";
import { getApiUrl } from "@/hooks/useConfig";

type MeResponse = {
  user_id: string;
  email: string;
  role: "admin" | "user" | "guest";
  auth_provider: string;
};

export const meQueryOptions = () =>
  queryOptions({
    queryKey: ["auth", "me"] as const,
    queryFn: async (): Promise<Session | null> => {
      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/auth/me`, {
        credentials: "include",
      });

      if (response.status === 401) return null;

      if (response.status === 403) {
        const body = (await response.json().catch(() => null)) as {
          error?: Array<string>;
        } | null;
        const code = body?.error?.[0];
        if (
          code === "MFA_SETUP_REQUIRED" &&
          !window.location.pathname.startsWith("/auth/mfa/setup-required")
        ) {
          window.location.href = "/auth/mfa/setup-required";
        }
        return null;
      }

      if (!response.ok) throw new Error(`/auth/me failed: ${response.status}`);

      const data = (await response.json()) as MeResponse;
      return {
        userId: data.user_id,
        email: data.email,
        role: data.role,
        authProvider: data.auth_provider,
      };
    },
    staleTime: 5 * 60 * 1000,
    retry: 1,
    refetchOnWindowFocus: true,
  });
