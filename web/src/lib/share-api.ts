import type { RequestParams } from "@/lib/api";
import { getApiUrl } from "@/hooks/useConfig";
import {
  MAX_RATE_LIMIT_RETRIES,
  buildUrlWithParams,
  getRetryAfterMs,
  sleep,
} from "@/lib/api";

type ShareRequestOptions = {
  method?: string;
  body?: object;
  params?: RequestParams;
  retryOnRateLimit?: boolean;
};

export async function shareFetch<T>(
  path: string,
  options: ShareRequestOptions = {},
  rateLimitAttempt = 0,
): Promise<T> {
  const { method = "GET", body, params } = options;
  const url = buildUrlWithParams(`${getApiUrl()}/shares${path}`, params);

  const response = await fetch(url, {
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const res = await response.json();
    const errorCode = res.error?.[0];

    if (
      response.status === 429 &&
      errorCode === "RATE_LIMIT_EXCEEDED" &&
      options.retryOnRateLimit !== false
    ) {
      if (rateLimitAttempt < MAX_RATE_LIMIT_RETRIES) {
        await sleep(getRetryAfterMs(response, rateLimitAttempt));
        return shareFetch<T>(path, options, rateLimitAttempt + 1);
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
