import i18n from "i18next";
import { toast } from "sonner";
import type { IDownloadFileResponse } from "@/components/bucket-view/helpers/types";
import { api } from "@/lib/api";


export const api_downloadFile = (
  bucketId: string,
  fileId: string,
  context?: "preview" | "download",
) =>
  api.get<IDownloadFileResponse>(`/buckets/${bucketId}/files/${fileId}/url`, {
    params: { context },
  });

export const downloadFromStorage = (url: string, filename: string) => {
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.rel = "noopener";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();

  toast.success(i18n.t("common.success"), {
    description: i18n.t("toast.download_started", { filename }),
  });
};
