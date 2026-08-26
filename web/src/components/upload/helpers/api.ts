import type { IUploadPresign } from "@/components/upload/helpers/types";
import type { IFolder } from "@/types/folder";
import { api } from "@/lib/api";

export const api_createFile = (
  name: string,
  bucketId: string,
  size: number,
  folderId: string | undefined,
  expiresAt: string | null,
) =>
  api.post<IUploadPresign>(`/buckets/${bucketId}/files`, {
    name,
    size,
    folder_id: folderId,
    expires_at: expiresAt,
  });

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

  await api.post(`/buckets/${bucketId}/files/trash`, {
    folder_ids: isFolder ? [fileId] : [],
    file_ids: isFolder ? [] : [fileId],
  });

  return { filename };
};
