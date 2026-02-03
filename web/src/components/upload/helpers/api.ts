import i18n from "i18next";
import type { ICreateFile } from "@/components/upload/helpers/types";
import type { IFolder } from "@/types/folder";
import { api } from "@/lib/api";

import { toast } from "@/components/ui/hooks/use-toast";

export const api_createFile = (
  name: string,
  bucketId: string,
  size: number,
  folderId: string | undefined,
) =>
  api.post<ICreateFile>(`/buckets/${bucketId}/files`, {
    name,
    size,
    folder_id: folderId,
  });

export const uploadToStorage = async (
  presignedUpload: ICreateFile,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> => {
  const xhr = new XMLHttpRequest();

  return new Promise((resolve, reject) => {
    if (signal) {
      signal.addEventListener("abort", () => {
        xhr.abort();
        reject(new Error("Upload cancelled"));
      });
    }

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        onProgress(Math.round((event.loaded / event.total) * 100));
      }
    });

    xhr.addEventListener("loadend", () => {
      if (xhr.readyState === 4) {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(`Upload failed with status ${xhr.status}`));
        }
      }
    });

    xhr.addEventListener("error", () => {
      reject(new Error("Network error during upload"));
    });

    xhr.addEventListener("abort", () => {
      reject(new Error("Upload cancelled"));
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
      title: i18n.t("toast.uploading"),
      description: i18n.t("toast.upload_started", { filename: file.name }),
    });
  });
};

export const createFolderMutationFn = async (params: {
  name: string;
  folderId: string | undefined;
  bucketId: string;
}): Promise<IFolder> => {
  const { name, folderId, bucketId } = params;
  return api.post<IFolder>(`/buckets/${bucketId}/folders`, {
    name,
    folder_id: folderId,
  });
};

export const api_confirmUpload = async (
  bucketId: string,
  fileId: string,
): Promise<void> => {
  await api.patch(`/buckets/${bucketId}/files/${fileId}`, {
    status: "uploaded",
  });
};

export const deleteFileMutationFn = async (params: {
  bucketId: string;
  fileId: string;
  filename?: string;
  isFolder?: boolean;
}): Promise<{ filename?: string }> => {
  const { bucketId, fileId, filename, isFolder = false } = params;

  if (isFolder) {
    await api.patch(`/buckets/${bucketId}/folders/${fileId}`, {
      status: "deleted",
    });
  } else {
    await api.patch(`/buckets/${bucketId}/files/${fileId}`, {
      status: "deleted",
    });
  }

  return { filename };
};
