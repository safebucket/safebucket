import React from "react";

import {
  AlertCircle,
  CircleCheck,
  Clock,
  FileIcon,
  Upload,
} from "lucide-react";

import type { UploadStatus } from "./types";

export const getStatusIcon = (
  status: UploadStatus,
  progress: number,
): React.JSX.Element => {
  switch (status) {
    case "success":
      return <CircleCheck className="h-5 w-5 text-green-500" />;
    case "error":
      return <AlertCircle className="h-5 w-5 text-red-500" />;
    case "uploading":
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
  t?: (key: string) => string,
) => {
  switch (status) {
    case "success":
      return t ? t("upload.status.completed") : "Completed";
    case "error":
      return t ? t("upload.status.failed") : "Failed";
    case "uploading":
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
    case "success":
      return "bg-green-500";
    case "error":
      return "bg-red-500";
    case "uploading":
      return "bg-blue-500";
    default:
      return "bg-gray-500";
  }
};
