import { queryOptions, useMutation } from "@tanstack/react-query";
import type {
  IFileTransferResponse,
  IPublicShareResponse,
  IShareUploadBody,
} from "@/types/share";
import { getApiUrl } from "@/hooks/useConfig";

async function shareFetch<T>(
  path: string,
  options: { method?: string; body?: object } = {},
): Promise<T> {
  const { method = "GET", body } = options;
  const url = `${getApiUrl()}/shares${path}`;

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

export const shareContentQueryOptions = (shareId: string) =>
  queryOptions({
    queryKey: ["shares", shareId, "content"],
    queryFn: () => shareFetch<IPublicShareResponse>(`/${shareId}/`),
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
      shareFetch<null>(`/${shareId}/auth`, {
        method: "POST",
        body: { password },
      }),
  });

export const useShareDownloadMutation = (shareId: string) =>
  useMutation({
    mutationFn: (fileId: string) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files/${fileId}`),
  });

export const useShareUploadMutation = (shareId: string) =>
  useMutation({
    mutationFn: (body: IShareUploadBody) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files`, {
        method: "POST",
        body,
      }),
  });

export const useShareConfirmUploadMutation = (shareId: string) =>
  useMutation({
    mutationFn: (fileId: string) =>
      shareFetch<null>(`/${shareId}/files/${fileId}`, {
        method: "PATCH",
      }),
  });
