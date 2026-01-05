import { getApiUrl } from "@/hooks/useConfig.ts";
import {
  authCookies,
  logout as authLogout,
  refreshAccessToken,
} from "@/lib/auth-service";

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

  const authHeader: Record<string, string> = {};
  if (!headers["Authorization"] && token) {
    authHeader["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(fullUrl, {
    method,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...authHeader,
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 403) {
    const res = await response.json();
    const errorCode = res.error?.[0];

    // Handle MFA setup required - redirect to MFA setup page (if not already there)
    if (errorCode === "MFA_SETUP_REQUIRED") {
      if (!window.location.pathname.startsWith("/auth/mfa/setup-required")) {
        window.location.href = "/auth/mfa/setup-required";
      }
      throw new Error(errorCode);
    }

    // Try to refresh token for other 403 errors
    if (retry) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        return fetchApi<T>(url, options, false);
      } else {
        authLogout();
      }
    }
    throw new Error(errorCode);
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
  delete(url: string, body?: object, options?: RequestOptions): Promise<null> {
    return fetchApi(url, { ...options, method: "DELETE", body });
  },
};
