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

  const response = await fetch(fullUrl, {
    method,
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      ...headers,
      // ...(cookie ? { Cookie: cookie } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    // credentials: "include",
    // cache,
    // next,
  });

  // TODO(YLB): Hook for notifications
  // if (!response.ok) {
  //   const message = (await response.json()).message || response.statusText;
  //   if (typeof window !== "undefined") {
  //     useNotifications.getState().addNotification({
  //       type: "error",
  //       title: "Error",
  //       message,
  //     });
  //   }
  //   throw new Error(message);
  // }

  if (
    response.status === 204 ||
    response.headers.get("Content-Length") === "0"
  ) {
    return null as T;
  }

  return response.json();
}

export const api = {
  get<T>(url: string, options?: RequestOptions): Promise<T | null> {
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
  patch<T>(
    url: string,
    body?: object,
    options?: RequestOptions,
  ): Promise<T | null> {
    return fetchApi<T>(url, { ...options, method: "PATCH", body });
  },
  delete(url: string, options?: RequestOptions): Promise<null> {
    return fetchApi(url, { ...options, method: "DELETE" });
  },
};
