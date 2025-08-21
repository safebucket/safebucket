import React from "react";
import { CircleCheck, FileIcon, Upload, AlertCircle, Clock } from "lucide-react";
import { UploadStatus } from "./types";

export const getStatusIcon = (status: UploadStatus, progress: number): React.JSX.Element => {
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
      return <FileIcon className="h-5 w-5 text-muted-foreground" />;
  }
};

export const getStatusText = (status: UploadStatus, progress: number) => {
  switch (status) {
    case UploadStatus.success:
      return "Completed";
    case UploadStatus.failed:
      return "Failed";
    case UploadStatus.uploading:
      if (progress === 0) {
        return "Preparing...";
      }
      return `${progress}%`;
    default:
      return "Unknown";
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