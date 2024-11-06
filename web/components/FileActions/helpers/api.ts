import { api } from "@/lib/api";

export const api_deleteFile = (fileId: string) =>
  api.delete(`/files/${fileId}`);
