import { api } from "@/lib/api";

import { toast } from "@/components/hooks/use-toast";
import { ICreateFile } from "@/components/upload/helpers/types";

export const api_createFile = (name: string, bucket_id?: string) =>
  api.post<ICreateFile>("/files", { name, bucket_id });

export default async function uploadToStorage(
  url: string,
  file: File,
  transferId: string,
  setProgress: (transferId: string, progress: number) => void,
): Promise<boolean> {
  const xhr = new XMLHttpRequest();
  return await new Promise((resolve) => {
    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        setProgress(transferId, Math.round((event.loaded / event.total) * 100));
      }
    });

    xhr.addEventListener("loadend", () => {
      resolve(xhr.readyState === 4 && xhr.status === 200);
    });

    xhr.open("PUT", url, true);
    // xhr.setRequestHeader("Content-Type", "application/octet-stream");
    xhr.send(file);

    toast({
      variant: "success",
      title: "Uploading",
      description: `Upload started for ${file.name}`,
    });
  });
}
