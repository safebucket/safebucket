import { queryOptions, useMutation } from "@tanstack/react-query";
import type {
  IPublicShareResponse,
  IShareDownloadArgs,
  IShareDownloadResponse,
  IShareUploadBody,
} from "@/types/share";
import type { IUploadPresign } from "@/components/upload/helpers/types";
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
      shareFetch<IShareDownloadResponse>(`/${shareId}/files/${fileId}/url`, {
        params: { context },
      }),
  });

export const useShareUploadMutation = (shareId: string) =>
  useMutation({
    meta: { skipGlobalErrorToast: true },
    mutationFn: (body: IShareUploadBody) =>
      shareFetch<IUploadPresign>(`/${shareId}/files`, {
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
