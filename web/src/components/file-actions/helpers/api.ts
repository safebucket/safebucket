import i18n from "i18next";
import type { IDownloadFileResponse } from "@/components/bucket-view/helpers/types";
import { api } from "@/lib/api";

import { toast } from "@/components/ui/hooks/use-toast";

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

  toast({
    variant: "success",
    title: i18n.t("common.success"),
    description: i18n.t("toast.download_started", { filename }),
  });
};
