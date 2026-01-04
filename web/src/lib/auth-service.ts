import Cookies from "js-cookie";
import { jwtDecode } from "jwt-decode";

import type {
  IJWTPayload,
  ILoginForm,
  ILoginResponse,
  IMFADevice,
  Session,
} from "@/components/auth-view/types/session";
import { getApiUrl } from "@/hooks/useConfig";
import { api } from "@/lib/api";

const COOKIE_ACCESS_TOKEN = "safebucket_access_token";
const COOKIE_REFRESH_TOKEN = "safebucket_refresh_token";
const COOKIE_AUTH_PROVIDER = "safebucket_auth_provider";

/**
 * Auth Service - Single source of truth for authentication state
 * All cookie management and auth logic centralized here
 */

export const authCookies = {
  getAccessToken: (): string | undefined => {
    return Cookies.get(COOKIE_ACCESS_TOKEN);
  },

  getRefreshToken: (): string | undefined => {
    return Cookies.get(COOKIE_REFRESH_TOKEN);
  },

  getAuthProvider: (): string | undefined => {
    return Cookies.get(COOKIE_AUTH_PROVIDER);
  },

  setAccessToken: (token: string): void => {
    Cookies.set(COOKIE_ACCESS_TOKEN, token, {
      secure: true,
      sameSite: "strict",
      path: "/",
    });
  },

  setRefreshToken: (token: string): void => {
    Cookies.set(COOKIE_REFRESH_TOKEN, token, {
      secure: true,
      sameSite: "strict",
      path: "/",
    });
  },

  setAuthProvider: (provider: string): void => {
    Cookies.set(COOKIE_AUTH_PROVIDER, provider, {
      secure: true,
      sameSite: "strict",
      path: "/",
    });
  },

  clearAll: (): void => {
    Cookies.remove(COOKIE_ACCESS_TOKEN);
    Cookies.remove(COOKIE_REFRESH_TOKEN);
    Cookies.remove(COOKIE_AUTH_PROVIDER);
  },

  setAll: (
    accessToken: string,
    refreshToken: string,
    provider: string,
  ): void => {
    authCookies.setAccessToken(accessToken);
    authCookies.setRefreshToken(refreshToken);
    authCookies.setAuthProvider(provider);
  },
};

export interface DecodedToken {
  payload: IJWTPayload;
  isExpired: boolean;
  expiresAt: Date;
}

/**
 * Safely decode JWT token with error handling
 */
// Token expiry buffer in milliseconds (30 seconds)
// Tokens are considered expired 30s before actual expiry to prevent race conditions
const TOKEN_EXPIRY_BUFFER_MS = 30000;

export const decodeToken = (token: string): DecodedToken | null => {
  try {
    const payload = jwtDecode<IJWTPayload>(token);
    const expiresAt = new Date(payload.exp * 1000);
    // Add buffer: consider token expired 30s before actual expiry
    const isExpired = Date.now() >= payload.exp * 1000 - TOKEN_EXPIRY_BUFFER_MS;

    return {
      payload,
      isExpired,
      expiresAt,
    };
  } catch (error) {
    console.error("Failed to decode JWT token:", error);
    return null;
  }
};

/**
 * Get current session from cookies
 * Note: Tokens are kept in cookies only for security, not exposed in session object
 */
export const getCurrentSession = (): Session | null => {
  const accessToken = authCookies.getAccessToken();
  const authProvider = authCookies.getAuthProvider();

  if (!accessToken || !authProvider) {
    return null;
  }

  const decoded = decodeToken(accessToken);
  if (!decoded || decoded.isExpired) {
    return null;
  }

  return {
    userId: decoded.payload.user_id,
    email: decoded.payload.email,
    role: decoded.payload.role,
    authProvider,
  };
};

export const loginWithProvider = (provider: string): void => {
  const apiUrl = getApiUrl();
  window.location.href = `${apiUrl}/auth/providers/${provider}/begin`;
};

export interface LoginResult {
  success: boolean;
  error?: string;
  mfaRequired?: boolean;
  mfaToken?: string;
  mfaSetupRequired?: boolean;
  devices?: IMFADevice[];
  userId?: string;
}

export const loginWithCredentials = async (
  credentials: ILoginForm,
): Promise<LoginResult> => {
  try {
    const response = await api.post<ILoginResponse>("/auth/login", credentials);

    // Check if MFA is required
    if (response.mfa_required && response.mfa_token) {
      // Decode MFA token to get user ID
      const decoded = decodeToken(response.mfa_token);
      return {
        success: false,
        mfaRequired: true,
        mfaToken: response.mfa_token,
        devices: response.devices,
        userId: decoded?.payload.user_id,
      };
    }

    // Check if MFA setup is required (admin enforced but not set up)
    // Note: No access/refresh tokens are issued until MFA setup is complete
    if (response.mfa_setup_required && response.mfa_token) {
      // Decode MFA token to get user ID
      const decoded = decodeToken(response.mfa_token);
      return {
        success: false,
        mfaSetupRequired: true,
        mfaToken: response.mfa_token,
        userId: decoded?.payload.user_id,
      };
    }

    // Normal login success
    if (response.access_token && response.refresh_token) {
      authCookies.setAll(
        response.access_token,
        response.refresh_token,
        "local",
      );
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Login failed",
    };
  }
};

/**
 * Verify MFA code during login
 * @param mfaToken - The MFA token from login response
 * @param code - The TOTP code from authenticator app
 * @param deviceId - Optional device ID to verify against (uses primary if not provided)
 */
export const verifyMFALogin = async (
  mfaToken: string,
  code: string,
  deviceId?: string,
): Promise<{ success: boolean; error?: string }> => {
  try {
    const response = await api.post<ILoginResponse>("/auth/mfa/verify", {
      mfa_token: mfaToken,
      code,
      ...(deviceId && { device_id: deviceId }),
    });

    if (response.access_token && response.refresh_token) {
      authCookies.setAll(
        response.access_token,
        response.refresh_token,
        "local",
      );
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "MFA verification failed",
    };
  }
};

export const logout = (): void => {
  authCookies.clearAll();
};

// Token refresh queue - prevents duplicate refresh calls
// Single-flight pattern: only one refresh at a time
let refreshPromise: Promise<boolean> | null = null;

/**
 * Refresh the access token using the refresh token
 * Returns true if refresh succeeded, false otherwise
 * Uses single-flight pattern to prevent duplicate refresh calls
 */
export const refreshAccessToken = async (): Promise<boolean> => {
  // If refresh is already in progress, return the existing promise
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const refreshToken = authCookies.getRefreshToken();

      if (!refreshToken) {
        return false;
      }

      const apiUrl = getApiUrl();
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 10000);

      const response = await fetch(`${apiUrl}/auth/refresh`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          refresh_token: refreshToken,
        }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        return false;
      }

      const data = await response.json();
      const newToken = data.access_token;

      if (newToken) {
        authCookies.setAccessToken(newToken);
        return true;
      }

      return false;
    } catch (err) {
      return false;
    } finally {
      // Clear the promise after completion
      refreshPromise = null;
    }
  })();

  return refreshPromise;
};

/**
 * Get current session with automatic token refresh if expired
 * Use this on app initialization to handle expired tokens gracefully
 */
export const getCurrentSessionWithRefresh =
  async (): Promise<Session | null> => {
    const accessToken = authCookies.getAccessToken();
    const authProvider = authCookies.getAuthProvider();

    if (!accessToken || !authProvider) {
      return null;
    }

    const decoded = decodeToken(accessToken);

    // If token is expired, try to refresh it
    if (decoded && decoded.isExpired) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        // Get the new token and decode it
        const newAccessToken = authCookies.getAccessToken();
        if (newAccessToken) {
          const newDecoded = decodeToken(newAccessToken);
          if (newDecoded && !newDecoded.isExpired) {
            return {
              userId: newDecoded.payload.user_id,
              email: newDecoded.payload.email,
              role: newDecoded.payload.role,
              authProvider,
            };
          }
        }
      }
      return null;
    }

    if (!decoded) {
      return null;
    }

    return {
      userId: decoded.payload.user_id,
      email: decoded.payload.email,
      role: decoded.payload.role,
      authProvider,
    };
  };
