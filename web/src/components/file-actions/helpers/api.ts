import i18n from "i18next";
import type { IDownloadFileResponse } from "@/components/bucket-view/helpers/types";
import { api } from "@/lib/api";
import { triggerBlobDownload } from "@/lib/download";

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
  const xhr = new XMLHttpRequest();

  xhr.onreadystatechange = () => {
    if (xhr.readyState === 4 && xhr.status === 200) {
      triggerBlobDownload(xhr.response, filename);
    }
  };
  xhr.responseType = "blob";
  xhr.open("GET", url, true);
  xhr.send(null);

  toast({
    variant: "success",
    title: i18n.t("common.success"),
    description: i18n.t("toast.download_started", { filename }),
  });
};
