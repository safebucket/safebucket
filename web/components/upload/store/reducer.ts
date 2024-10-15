import {
  ADD_TRANSFER,
  UPDATE_PROGRESS,
  UPDATE_STATUS,
} from "@/components/upload/helpers/constants";
import { IUpload, UploadStatus } from "@/components/upload/helpers/types";
import { TransferAction } from "@/components/upload/store/index";

export const transfersReducer = (
  transfers: IUpload[],
  action: TransferAction,
) => {
  switch (action.type) {
    case ADD_TRANSFER: {
      const upload: IUpload = {
        id: action.payload.id,
        name: action.payload.name,
        progress: 0,
        status: UploadStatus.uploading,
      };

      return [...transfers, upload];
    }
    case UPDATE_PROGRESS: {
      return transfers.map((transfer: IUpload) => {
        if (transfer.id === action.payload.id) {
          return { ...transfer, progress: action.payload.progress };
        }
        return transfer;
      });
    }
    case UPDATE_STATUS: {
      return transfers.map((transfer: IUpload) => {
        if (transfer.id === action.payload.id) {
          return { ...transfer, status: action.payload.status };
        }
        return transfer;
      });
    }
    default: {
      return transfers;
    }
  }
};
