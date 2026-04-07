import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type {
  IFileTransferResponse,
  IPublicShareResponse,
  IShareAuthResponse,
  IShareUploadBody,
} from "@/types/share";
import { getApiUrl } from "@/hooks/useConfig";

async function shareFetch<T>(
  path: string,
  options: { method?: string; body?: object; token?: string } = {},
): Promise<T> {
  const { method = "GET", body, token } = options;
  const url = `${getApiUrl()}/shares${path}`;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    method,
    headers,
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

export const shareContentQueryOptions = (
  shareId: string,
  token: string | null,
) =>
  queryOptions({
    queryKey: ["shares", shareId, "content", token],
    queryFn: () =>
      shareFetch<IPublicShareResponse>(`/${shareId}/`, {
        token: token ?? undefined,
      }),
    enabled: false,
  });

export const useShareAuthMutation = () =>
  useMutation({
    mutationFn: ({
      shareId,
      password,
    }: {
      shareId: string;
      password: string;
    }) =>
      shareFetch<IShareAuthResponse>(`/${shareId}/auth`, {
        method: "POST",
        body: { password },
      }),
  });

export const useShareDownloadMutation = (
  shareId: string,
  token: string | null,
) =>
  useMutation({
    mutationFn: (fileId: string) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files/${fileId}`, {
        token: token ?? undefined,
      }),
  });

export const useShareUploadMutation = (
  shareId: string,
  token: string | null,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: IShareUploadBody) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files`, {
        method: "POST",
        body,
        token: token ?? undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["shares", shareId, "content"],
      });
    },
  });
};

export const useShareConfirmUploadMutation = (
  shareId: string,
  token: string | null,
) =>
  useMutation({
    mutationFn: (fileId: string) =>
      shareFetch<null>(`/${shareId}/files/${fileId}`, {
        method: "PATCH",
        token: token ?? undefined,
      }),
  });
