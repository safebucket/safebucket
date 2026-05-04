import { useCallback } from "react";
import { useRouteContext, useRouter } from "@tanstack/react-router";

import type { ILoginForm, Session } from "@/components/auth-view/types/session";
import type { LoginResult } from "@/lib/auth-service";
import {
  logout as authLogout,
  verifyMFALogin as authVerifyMFA,
  getCurrentSession,
  loginWithCredentials,
  loginWithProvider,
} from "@/lib/auth-service";

export function useSession(): Session | null {
  const context = useRouteContext({ from: "__root__" });
  return context.session;
}

export function useLogin() {
  const router = useRouter();
  const { queryClient } = useRouteContext({ from: "__root__" });

  const loginOAuth = useCallback((provider: string) => {
    loginWithProvider(provider);
  }, []);

  const loginLocal = useCallback(
    async (credentials: ILoginForm): Promise<LoginResult> => {
      const result = await loginWithCredentials(credentials);

      if (result.success && !result.mfaRequired) {
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

export function useLogout() {
  const router = useRouter();
  const { queryClient } = useRouteContext({ from: "__root__" });

  return useCallback(async () => {
    await authLogout();

    router.update({
      context: {
        queryClient,
        session: null,
      },
    });

    router.navigate({ to: "/auth/login", search: { redirect: undefined } });
  }, [router, queryClient]);
}

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
