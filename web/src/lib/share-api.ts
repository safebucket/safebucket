import type { RequestParams } from "@/lib/api";
import { getApiUrl } from "@/hooks/useConfig";
import { buildUrlWithParams } from "@/lib/api";

type ShareRequestOptions = {
  method?: string;
  body?: object;
  params?: RequestParams;
};

export async function shareFetch<T>(
  path: string,
  options: ShareRequestOptions = {},
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
    throw new Error(res.error?.[0] ?? "INTERNAL_SERVER_ERROR");
  }

  if (
    response.status === 204 ||
    response.headers.get("Content-Length") === "0"
  ) {
    return null as T;
  }

  return response.json();
}
