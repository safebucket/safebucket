import type {
  ILoginForm,
  ILoginResponse,
  Session,
} from "@/components/auth-view/types/session";
import { getApiUrl } from "@/hooks/useConfig";
import { api } from "@/lib/api";

export const loginWithProvider = (provider: string): void => {
  const apiUrl = getApiUrl();
  window.location.href = `${apiUrl}/auth/providers/${provider}/begin`;
};

export interface LoginResult {
  success: boolean;
  error?: string;
  mfaRequired?: boolean;
}

export const loginWithCredentials = async (
  credentials: ILoginForm,
): Promise<LoginResult> => {
  try {
    const response = await api.post<ILoginResponse>("/auth/login", credentials);

    if (response.mfa_required) {
      return { success: false, mfaRequired: true };
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Login failed",
    };
  }
};

export const loginWithLDAP = async (
  provider: string,
  credentials: ILoginForm,
): Promise<LoginResult> => {
  try {
    const response = await api.post<ILoginResponse>(
      `/auth/providers/${provider}/login`,
      credentials,
    );

    if (response.mfa_required) {
      return { success: false, mfaRequired: true };
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Login failed",
    };
  }
};

export const verifyMFALogin = async (
  code: string,
  deviceId?: string,
): Promise<{ success: boolean; error?: string }> => {
  try {
    await api.post<ILoginResponse>("/auth/mfa/verify", {
      code,
      ...(deviceId && { device_id: deviceId }),
    });

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "MFA verification failed",
    };
  }
};

let refreshPromise: Promise<boolean> | null = null;

export const refreshAccessToken = async (): Promise<boolean> => {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const apiUrl = getApiUrl();
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 10000);

      const response = await fetch(`${apiUrl}/auth/refresh`, {
        method: "POST",
        credentials: "include",
        signal: controller.signal,
      });

      clearTimeout(timeoutId);
      return response.ok;
    } catch {
      return false;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
};

async function fetchMeSession(): Promise<Session | null> {
  try {
    const apiUrl = getApiUrl();
    const response = await fetch(`${apiUrl}/auth/me`, {
      credentials: "include",
    });
    if (!response.ok) return null;
    const data = await response.json();
    return {
      userId: data.user_id,
      email: data.email,
      role: data.role,
      authProvider: data.auth_provider,
    };
  } catch {
    return null;
  }
}

export const getCurrentSessionWithRefresh =
  async (): Promise<Session | null> => {
    const session = await fetchMeSession();
    if (session) return session;

    const refreshed = await refreshAccessToken();
    if (refreshed) return fetchMeSession();

    return null;
  };
