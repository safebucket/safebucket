import { useCallback } from "react";
import { useRouteContext, useRouter } from "@tanstack/react-router";

import type { ILoginForm, Session } from "@/components/auth-view/types/session";
import {
  getCurrentSession,
  loginWithCredentials,
  loginWithProvider,
  logout as authLogout,
  verifyMFALogin as authVerifyMFA,
  type LoginResult,
} from "@/lib/auth-service";

/**
 * Hook to access current session from router context
 * Use this in components to get session state
 */
export function useSession(): Session | null {
  const context = useRouteContext({ from: "__root__" });
  return context.session;
}

/**
 * Hook to handle login (both OAuth and credentials)
 */
export function useLogin() {
  const router = useRouter();
  const { queryClient } = useRouteContext({ from: "__root__" });

  const loginOAuth = useCallback((provider: string) => {
    // OAuth redirects to external provider
    loginWithProvider(provider);
  }, []);

  const loginLocal = useCallback(
    async (credentials: ILoginForm): Promise<LoginResult> => {
      const result = await loginWithCredentials(credentials);

      if (result.success && !result.mfaRequired) {
        // Update router context with new session
        // This includes mfaSetupRequired case where tokens are set but MFA setup is needed
        const session = getCurrentSession();
        router.update({
          context: {
            queryClient,
            session,
          },
        });
      }

      return result;
    },
    [router, queryClient],
  );

  const verifyMFA = useCallback(
    async (
      mfaToken: string,
      code: string,
      deviceId?: string,
    ): Promise<{ success: boolean; error?: string }> => {
      const result = await authVerifyMFA(mfaToken, code, deviceId);

      if (result.success) {
        // Update router context with new session
        const session = getCurrentSession();
        router.update({
          context: {
            queryClient,
            session,
          },
        });
      }

      return result;
    },
    [router, queryClient],
  );

  return {
    loginOAuth,
    loginLocal,
    verifyMFA,
  };
}

/**
 * Hook to handle logout
 */
export function useLogout() {
  const router = useRouter();
  const { queryClient } = useRouteContext({ from: "__root__" });

  return useCallback(() => {
    authLogout();

    router.update({
      context: {
        queryClient,
        session: null,
      },
    });

    router.navigate({ to: "/auth/login", search: { redirect: undefined } });
  }, [router, queryClient]);
}

/**
 * Hook to refresh session from cookies
 * Use after OAuth callback, password reset, invite acceptance
 */
export function useRefreshSession() {
  const router = useRouter();
  const { queryClient } = useRouteContext({ from: "__root__" });

  return useCallback(() => {
    const session = getCurrentSession();

    router.update({
      context: {
        queryClient,
        session,
      },
    });
  }, [router, queryClient]);
}
