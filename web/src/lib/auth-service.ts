import Cookies from "js-cookie";
import { jwtDecode } from "jwt-decode";

import type {
  IJWTPayload,
  ILoginForm,
  ILoginResponse,
  Session,
} from "@/components/auth-view/types/session";
import { getApiUrl } from "@/hooks/useConfig";
import { api } from "@/lib/api";

// Cookie names as constants
const COOKIE_ACCESS_TOKEN = "safebucket_access_token";
const COOKIE_REFRESH_TOKEN = "safebucket_refresh_token";
const COOKIE_AUTH_PROVIDER = "safebucket_auth_provider";

/**
 * Auth Service - Single source of truth for authentication state
 * All cookie management and auth logic centralized here
 */

// ============================================================================
// Cookie Management
// ============================================================================

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

// ============================================================================
// JWT Token Utilities
// ============================================================================

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
 * Check if current session is valid based on stored tokens
 */
export const isAuthenticated = (): boolean => {
  const accessToken = authCookies.getAccessToken();
  const authProvider = authCookies.getAuthProvider();

  if (!accessToken || !authProvider) {
    return false;
  }

  const decoded = decodeToken(accessToken);
  if (!decoded || decoded.isExpired) {
    return false;
  }

  return true;
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

// ============================================================================
// Authentication Actions
// ============================================================================

/**
 * OAuth provider login - redirects to backend OAuth flow
 */
export const loginWithProvider = (provider: string): void => {
  const apiUrl = getApiUrl();
  window.location.href = `${apiUrl}/auth/providers/${provider}/begin`;
};

/**
 * Local email/password login
 */
export const loginWithCredentials = async (
  credentials: ILoginForm,
): Promise<{ success: boolean; error?: string }> => {
  try {
    const response = await api.post<ILoginResponse>("/auth/login", credentials);

    authCookies.setAll(response.access_token, response.refresh_token, "local");

    return { success: true };
  } catch (error) {
    console.error("Login failed:", error);
    return {
      success: false,
      error: error instanceof Error ? error.message : "Login failed",
    };
  }
};

/**
 * Logout - clears all auth state
 */
export const logout = (): void => {
  authCookies.clearAll();
};

/**
 * Manually set authentication state
 * Used by password reset and invite acceptance flows
 */
export const setAuthenticationState = (
  accessToken: string,
  refreshToken: string,
  provider: string,
): void => {
  authCookies.setAll(accessToken, refreshToken, provider);
};
