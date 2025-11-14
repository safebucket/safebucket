import { getApiUrl } from "@/hooks/useConfig.ts";
import {
  authCookies,
  getCurrentSession,
  logout as authLogout,
} from "@/lib/auth-service";
import { queryClient } from "@/main";

type RequestOptions = {
  method?: string;
  headers?: Record<string, string>;
  body?: object;
  cookie?: string;
  params?: Record<string, string | number | boolean | undefined | null>;
};

function buildUrlWithParams(
  url: string,
  params?: RequestOptions["params"],
): string {
  if (!params) return url;
  const filteredParams = Object.fromEntries(
    Object.entries(params).filter(
      ([, value]) => value !== undefined && value !== null,
    ),
  );
  if (Object.keys(filteredParams).length === 0) return url;
  const queryString = new URLSearchParams(
    filteredParams as Record<string, string>,
  ).toString();
  return `${url}?${queryString}`;
}

export async function fetchApi<T>(
  url: string,
  options: RequestOptions = {},
  retry = true,
): Promise<T> {
  const { method = "GET", headers = {}, body, params } = options;
  const apiUrl = getApiUrl();
  const fullUrl = buildUrlWithParams(`${apiUrl}${url}`, params);

  const token = authCookies.getAccessToken();

  const response = await fetch(fullUrl, {
    method,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...headers,
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 403 && retry) {
    await refreshToken();
    return fetchApi<T>(url, options, false);
  }

  if (!response.ok) {
    const res = await response.json();
    throw new Error(res.error[0]);
  }

  if (
    response.status === 204 ||
    response.headers.get("Content-Length") === "0"
  ) {
    return null as T;
  }

  return response.json();
}

// Token refresh queue - prevents duplicate refresh calls
// Single-flight pattern: only one refresh at a time
let refreshPromise: Promise<void> | null = null;

async function refreshToken(): Promise<void> {
  // If refresh is already in progress, return the existing promise
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const refreshToken = authCookies.getRefreshToken();

      if (!refreshToken) {
        logout();
        return;
      }

      const body = JSON.stringify({
        refresh_token: refreshToken,
      });

      const apiUrl = getApiUrl();
      const response = await fetch(`${apiUrl}/auth/refresh`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body,
      });

      if (!response.ok) {
        logout();
        return;
      }

      const data = await response.json();
      const newToken = data.access_token;

      if (newToken) {
        authCookies.setAccessToken(newToken);

        // Update router context with refreshed session
        const session = getCurrentSession();
        if (window.router) {
          window.router.update({
            context: {
              queryClient,
              session,
            },
          });
        }
      } else {
        logout();
      }
    } catch (err) {
      logout();
    } finally {
      // Clear the promise after completion
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export function logout() {
  authLogout();
  window.location.href = "/auth/login";
}

export const api = {
  get<T>(url: string, options?: RequestOptions): Promise<T> {
    return fetchApi<T>(url, { ...options, method: "GET" });
  },
  post<T>(url: string, body?: object, options?: RequestOptions): Promise<T> {
    return fetchApi<T>(url, { ...options, method: "POST", body });
  },
  put<T>(
    url: string,
    body?: object,
    options?: RequestOptions,
  ): Promise<T | null> {
    return fetchApi<T>(url, { ...options, method: "PUT", body });
  },
  patch(url: string, body?: object, options?: RequestOptions): Promise<null> {
    return fetchApi(url, { ...options, method: "PATCH", body });
  },
  delete(url: string, options?: RequestOptions): Promise<null> {
    return fetchApi(url, { ...options, method: "DELETE" });
  },
};
