import {
  ADD_UPLOAD,
  UPDATE_PROGRESS,
  UPDATE_STATUS,
} from "@/components/upload/helpers/constants";
import { IUpload, UploadStatus } from "@/components/upload/helpers/types";
import { UploadAction } from "@/components/upload/store/index";

export const uploadsReducer = (uploads: IUpload[], action: UploadAction) => {
  switch (action.type) {
    case ADD_UPLOAD: {
      const upload: IUpload = {
        id: action.payload.id,
        name: action.payload.name,
        path: action.payload.path,
        progress: 0,
        status: UploadStatus.uploading,
      };

      return [...uploads, upload];
    }
    case UPDATE_PROGRESS: {
      return uploads.map((upload: IUpload) => {
        if (upload.id === action.payload.id) {
          return { ...upload, progress: action.payload.progress };
        }
        return upload;
      });
    }
    case UPDATE_STATUS: {
      return uploads.map((upload: IUpload) => {
        if (upload.id === action.payload.id) {
          return { ...upload, status: action.payload.status };
        }
        return upload;
      });
    }
    default: {
      return uploads;
    }
  }
};
