import React from "react";

import {
  AlertCircle,
  CircleCheck,
  Clock,
  FileIcon,
  Upload,
} from "lucide-react";

import { UploadStatus } from "./types";

export const getStatusIcon = (
  status: UploadStatus,
  progress: number,
): React.JSX.Element => {
  switch (status) {
    case UploadStatus.success:
      return <CircleCheck className="h-5 w-5 text-green-500" />;
    case UploadStatus.failed:
      return <AlertCircle className="h-5 w-5 text-red-500" />;
    case UploadStatus.uploading:
      if (progress === 0) {
        return <Clock className="h-5 w-5 text-blue-500" />;
      }
      return <Upload className="h-5 w-5 text-blue-500" />;
    default:
      return <FileIcon className="text-muted-foreground h-5 w-5" />;
  }
};

export const getStatusText = (
  status: UploadStatus, 
  progress: number, 
  t?: (key: string) => string
) => {
  switch (status) {
    case UploadStatus.success:
      return t ? t("upload.status.completed") : "Completed";
    case UploadStatus.failed:
      return t ? t("upload.status.failed") : "Failed";
    case UploadStatus.uploading:
      if (progress === 0) {
        return t ? t("upload.status.preparing") : "Preparing...";
      }
      return `${progress}%`;
    default:
      return t ? t("upload.status.unknown") : "Unknown";
  }
};

export const getProgressColor = (status: UploadStatus) => {
  switch (status) {
    case UploadStatus.success:
      return "bg-green-500";
    case UploadStatus.failed:
      return "bg-red-500";
    case UploadStatus.uploading:
      return "bg-blue-500";
    default:
      return "bg-gray-500";
  }
};
