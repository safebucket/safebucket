import { queryOptions, useMutation } from "@tanstack/react-query";
import type {
  IFileTransferResponse,
  IPublicShareResponse,
  IShareDownloadArgs,
  IShareUploadBody,
} from "@/types/share";
import { shareFetch } from "@/lib/share-api";

export const shareContentQueryOptions = (shareId: string) =>
  queryOptions({
    queryKey: ["shares", shareId, "content"],
    queryFn: () => shareFetch<IPublicShareResponse>(`/${shareId}/`),
    enabled: false,
  });

export const useShareAuthMutation = () =>
  useMutation({
    meta: { skipGlobalErrorToast: true },
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
        retryOnRateLimit: false,
      }),
  });

export const useShareDownloadMutation = (shareId: string) =>
  useMutation({
    mutationFn: ({ fileId, context }: IShareDownloadArgs) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files/${fileId}/url`, {
        params: { context },
      }),
  });

export const useShareUploadMutation = (shareId: string) =>
  useMutation({
    meta: { skipGlobalErrorToast: true },
    mutationFn: (body: IShareUploadBody) =>
      shareFetch<IFileTransferResponse>(`/${shareId}/files`, {
        method: "POST",
        body,
      }),
  });

export const useShareConfirmUploadMutation = (shareId: string) =>
  useMutation({
    meta: { skipGlobalErrorToast: true },
    mutationFn: (fileId: string) =>
      shareFetch<null>(`/${shareId}/files/${fileId}`, {
        method: "PATCH",
      }),
  });
