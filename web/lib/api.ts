import Cookies from "js-cookie";

type RequestOptions = {
  method?: string;
  headers?: Record<string, string>;
  body?: object;
  cookie?: string;
  params?: Record<string, string | number | boolean | undefined | null>;
  // cache?: RequestCache;
  // next?: NextFetchRequestConfig;
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
): Promise<T> {
  const {
    method = "GET",
    headers = {},
    body,
    params,
    // cookie,
    // cache = "no-store",
    // next,
  } = options;
  const fullUrl = buildUrlWithParams(
    `${process.env.NEXT_PUBLIC_API_URL}${url}`,
    params,
  );

  const token = Cookies.get("safebucket_access_token");

  const response = await fetch(fullUrl, {
    method,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...headers,
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    // credentials: "include",
    // cache,
    // next,
  });

  if (!response.ok) {
    const res = await response.json();
    throw new Error(res.error[0]);
  } else if (
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
  delete(url: string, options?: RequestOptions): Promise<null> {
    return fetchApi(url, { ...options, method: "DELETE" });
  },
};
