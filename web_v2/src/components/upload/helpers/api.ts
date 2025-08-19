import { api } from "@/lib/api";

import {
  FileType,
  IInviteResponse,
  IInvites,
} from "@/components/bucket-view/helpers/types";
import { toast } from "@/components/ui/hooks/use-toast";
import { ICreateFile } from "@/components/upload/helpers/types";

export const api_createFile = (
  name: string,
  type: FileType,
  path: string,
  bucketId: string,
  size?: number,
) =>
  api.post<ICreateFile>(`/buckets/${bucketId}/files`, {
    name,
    type,
    path,
    size,
  });

export const uploadToStorage = async (
  presignedUpload: ICreateFile,
  file: File,
  uploadId: string,
  setProgress: (uploadId: string, progress: number) => void,
): Promise<boolean> => {
  const xhr = new XMLHttpRequest();
  return await new Promise((resolve) => {
    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        setProgress(uploadId, Math.round((event.loaded / event.total) * 100));
      }
    });

    xhr.open("POST", presignedUpload.url, true);
    const formData = new FormData();
    Object.entries(presignedUpload.body).forEach(([key, value]) => {
      formData.append(key, value);
    });
    formData.append("file", file);

    xhr.send(formData);

    toast({
      variant: "success",
      title: "Uploading",
      description: `Upload started for ${file.name}`,
    });

    xhr.addEventListener("loadend", () => {
      resolve(xhr.readyState === 4 && xhr.status === 204);
    });
  });
};

export const api_updateMembers = (bucket_id: string, invites: IInvites[]) =>
  api.post<IInviteResponse[]>("/invites", { bucket_id, invites });

export const api_updateBucketName = (bucketId: string, name: string) =>
  api.patch(`/buckets/${bucketId}`, { name });

export const api_deleteBucket = (bucketId: string) =>
  api.delete(`/buckets/${bucketId}`);
