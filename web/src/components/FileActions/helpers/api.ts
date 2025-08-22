import type { IDownloadFileResponse } from "@/components/bucket-view/helpers/types";
import { api } from "@/lib/api";

import { toast } from "@/components/ui/hooks/use-toast";

export const api_downloadFile = (bucketId: string, fileId: string) =>
  api.get<IDownloadFileResponse>(
    `/buckets/${bucketId}/files/${fileId}/download`,
  );

export const downloadFromStorage = (url: string, filename: string) => {
  const xhr = new XMLHttpRequest();

  xhr.onreadystatechange = () => {
    if (xhr.readyState === 4 && xhr.status === 200) {
      const blobUrl = window.URL.createObjectURL(xhr.response);
      const e = document.createElement("a");
      e.href = blobUrl;
      e.download = filename;
      document.body.appendChild(e);
      e.click();
      document.body.removeChild(e);
    }
  };
  xhr.responseType = "blob";
  xhr.open("GET", url, true);
  xhr.send(null);

  toast({
    variant: "success",
    title: "Success",
    description: `Download started for file ${filename}`,
  });
};
