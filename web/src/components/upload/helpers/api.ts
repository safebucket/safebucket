import type { ICreateFile } from "@/components/upload/helpers/types";
import type { IFolder } from "@/types/folder";
import { api } from "@/lib/api";

import { toast } from "@/components/ui/hooks/use-toast";

export const api_createFile = (
  name: string,
  bucketId: string,
  size: number,
  folderId: string | null,
) =>
  api.post<ICreateFile>(`/buckets/${bucketId}/files`, {
    name,
    size,
    folder_id: folderId,
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

export const createFolderMutationFn = async (params: {
  name: string;
  folderId: string | null;
  bucketId: string;
}): Promise<IFolder> => {
  const { name, folderId, bucketId } = params;
  return api.post<IFolder>(`/buckets/${bucketId}/folders`, {
    name,
    folder_id: folderId,
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
    await api.delete(`/buckets/${bucketId}/folders/${fileId}`);
  } else {
    await api.delete(`/buckets/${bucketId}/files/${fileId}`);
  }

  return { filename };
};
