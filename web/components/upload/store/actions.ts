import {
  ADD_TRANSFER,
  UPDATE_PROGRESS,
  UPDATE_STATUS,
} from "@/components/upload/helpers/constants";
import { UploadStatus } from "@/components/upload/helpers/types";
import { TransferAction } from "@/components/upload/store/index";

function createAction(type: any, payload: any): TransferAction {
  return { type, payload };
}

export const addTransfer = (id: string, name: string) =>
  createAction(ADD_TRANSFER, { id, name });

export const updateProgress = (id: string, progress: number) =>
  createAction(UPDATE_PROGRESS, { id, progress });

export const updateStatus = (id: string, status: UploadStatus) =>
  createAction(UPDATE_STATUS, { id, status });
