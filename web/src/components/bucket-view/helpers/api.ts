import { api } from "@/lib/api";

export const api_restoreFile = (
  bucketId: string,
  fileId: string,
): Promise<null> =>
  api.post<null>(`/buckets/${bucketId}/trash/${fileId}/restore`);

export const api_purgeFile = (bucketId: string, fileId: string): Promise<null> =>
  api.delete(`/buckets/${bucketId}/trash/${fileId}`);
