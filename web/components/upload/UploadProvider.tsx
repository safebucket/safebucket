import React, { useReducer } from "react";

import uploadToStorage, {
  api_createFile,
} from "@/components/upload/helpers/api";
import {
  IStartUploadData,
  UploadStatus,
} from "@/components/upload/helpers/types";
import { UploadContext } from "@/components/upload/hooks/useUploadContext";
import { transfersReducer } from "@/components/upload/store/reducer";

import * as actions from "./store/actions";

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const [transfers, dispatch] = useReducer(transfersReducer, []);

  const addTransfer = (transferId: string, filename: string) =>
    dispatch(actions.addTransfer(transferId, filename));

  const updateProgress = (transferId: string, progress: number) =>
    dispatch(actions.updateProgress(transferId, progress));

  const updateStatus = (transferId: string, status: UploadStatus) =>
    dispatch(actions.updateStatus(transferId, status));

  const startUpload = async (data: IStartUploadData, bucketId?: string) => {
    const file = data.files[0];
    const transferId = crypto.randomUUID();

    addTransfer(transferId, file.name);

    api_createFile(file.name, bucketId).then(async (res) => {
      uploadToStorage(res.url, file, transferId, updateProgress).then(
        (success: boolean) => {
          const status = success ? UploadStatus.success : UploadStatus.failed;
          updateStatus(transferId, status);
        },
      );
    });
  };

  return (
    <UploadContext.Provider value={{ transfers, startUpload }}>
      {children}
    </UploadContext.Provider>
  );
};
