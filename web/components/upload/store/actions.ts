import {
  ADD_UPLOAD,
  UPDATE_PROGRESS,
  UPDATE_STATUS,
} from "@/components/upload/helpers/constants";
import { UploadStatus } from "@/components/upload/helpers/types";
import { UploadAction } from "@/components/upload/store/index";

const createAction = (type: any, payload: any): UploadAction => {
  return { type, payload };
};

export const addUpload = (id: string, name: string, path: string) =>
  createAction(ADD_UPLOAD, { id, name, path });

export const updateProgress = (id: string, progress: number) =>
  createAction(UPDATE_PROGRESS, { id, progress });

export const updateStatus = (id: string, status: UploadStatus) =>
  createAction(UPDATE_STATUS, { id, status });
