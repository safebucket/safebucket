import { getApiUrl } from "@/hooks/useConfig.ts";
import { refreshAccessToken } from "@/lib/auth-service";

export type RequestParams = Record<
  string,
  string | number | boolean | undefined | null
>;

type RequestOptions = {
  method?: string;
  headers?: Record<string, string>;
  body?: object;
  cookie?: string;
  params?: RequestParams;
  retryOnRateLimit?: boolean;
};

export const MAX_RATE_LIMIT_RETRIES = 3;

export const sleep = (ms: number) =>
  new Promise((resolve) => setTimeout(resolve, ms));

export function getRetryAfterMs(response: Response, attempt: number): number {
  const raw = parseInt(response.headers.get("Retry-After") ?? "", 10);
  const seconds =
    Number.isFinite(raw) && raw > 0 ? Math.min(raw, 60) : 2 ** attempt;
  return seconds * 1000 + Math.floor(Math.random() * 1000);
}

export function buildUrlWithParams(
  url: string,
  params?: RequestParams,
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
  rateLimitAttempt = 0,
): Promise<T> {
  const { method = "GET", headers = {}, body, params } = options;
  const apiUrl = getApiUrl();
  const fullUrl = buildUrlWithParams(`${apiUrl}${url}`, params);

  const response = await fetch(fullUrl, {
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let errorCode: string | undefined;
    try {
      const res = await response.json();
      errorCode = res.error?.[0];
    } catch {
      errorCode = undefined;
    }

    if (
      response.status === 429 &&
      errorCode === "RATE_LIMIT_EXCEEDED" &&
      options.retryOnRateLimit !== false
    ) {
      if (rateLimitAttempt < MAX_RATE_LIMIT_RETRIES) {
        await sleep(getRetryAfterMs(response, rateLimitAttempt));
        return fetchApi<T>(url, options, retry, rateLimitAttempt + 1);
      }
    }

    if (response.status === 401 && errorCode === "SESSION_REVOKED") {
      window.location.href = "/auth/login";
      throw new Error(errorCode);
    }

    if (response.status === 403) {
      if (errorCode === "MFA_SETUP_REQUIRED") {
        if (!window.location.pathname.startsWith("/auth/mfa/setup-required")) {
          window.location.href = "/auth/mfa/setup-required";
        }
        throw new Error(errorCode);
      }

      if (retry) {
        const refreshed = await refreshAccessToken();
        if (refreshed) {
          return fetchApi<T>(url, options, false);
        } else {
          await api.post("/auth/logout");
          window.location.href = "/auth/login";
        }
      }
    }

    throw new Error(errorCode ?? "INTERNAL_SERVER_ERROR");
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
